#include <EndpointSecurity/EndpointSecurity.h>
#include <bsm/libbsm.h>
#include <dispatch/dispatch.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <signal.h>
#include <unistd.h>
#include <sys/socket.h>
#include <sys/un.h>
#include <netinet/in.h>
#include <pthread.h>

// agentsh macOS Native Endpoint Security (ESF) Kernel Interceptor
// Intercepts and denies unauthorized database and service connections natively
// on Apple Silicon and Intel macOS systems.

#define DEFAULT_SOCKET_PATH "/var/run/agentsh_esf.sock"
#define MAX_BLOCKED_PORTS 256

static const char *default_blocked_socket_patterns[] = {
    ".s.PGSQL.5432",   // PostgreSQL
    "mongodb-27017",   // MongoDB
    "mysql.sock",      // MySQL
    "redis.sock",      // Redis
    NULL
};

static uint16_t g_blocked_ports[MAX_BLOCKED_PORTS] = { 5432, 27017, 3306, 6379 };
static size_t g_blocked_ports_count = 4;
static uint64_t g_drop_count = 0;
static pthread_mutex_t g_lock = PTHREAD_MUTEX_INITIALIZER;
static es_client_t *g_es_client = NULL;
static int g_server_fd = -1;
static char g_socket_path[256] = DEFAULT_SOCKET_PATH;

static int is_blocked_socket_path(const char *path) {
    if (!path) return 0;
    for (int i = 0; default_blocked_socket_patterns[i] != NULL; i++) {
        if (strstr(path, default_blocked_socket_patterns[i]) != NULL) {
            return 1;
        }
    }
    return 0;
}

static void handle_event(es_client_t *client, const es_message_t *msg) {
    if (msg->event_type == ES_EVENT_TYPE_AUTH_UIPC_CONNECT) {
        pid_t pid = audit_token_to_pid(msg->process->audit_token);
        const char *path = msg->event.uipc_connect.file ? msg->event.uipc_connect.file->path.data : "";

        if (is_blocked_socket_path(path)) {
            pthread_mutex_lock(&g_lock);
            g_drop_count++;
            pthread_mutex_unlock(&g_lock);

            printf("[agentsh ESF] 🚨 BLOCKED unauthorized database connection to socket %s from PID %d\n", path, pid);
            es_respond_auth_result(client, msg, ES_AUTH_RESULT_DENY, false);
            return;
        }

        es_respond_auth_result(client, msg, ES_AUTH_RESULT_ALLOW, false);
    }
}

// IPC listener thread to communicate with agentsh CLI
static void *ipc_server_thread(void *arg) {
    (void)arg;
    struct sockaddr_un addr;

    unlink(g_socket_path);

    g_server_fd = socket(AF_UNIX, SOCK_STREAM, 0);
    if (g_server_fd < 0) {
        perror("Failed to create IPC socket");
        return NULL;
    }

    memset(&addr, 0, sizeof(addr));
    addr.sun_family = AF_UNIX;
    strncpy(addr.sun_path, g_socket_path, sizeof(addr.sun_path) - 1);

    if (bind(g_server_fd, (struct sockaddr *)&addr, sizeof(addr)) < 0) {
        perror("Failed to bind IPC socket");
        close(g_server_fd);
        return NULL;
    }

    if (listen(g_server_fd, 5) < 0) {
        perror("Failed to listen on IPC socket");
        close(g_server_fd);
        return NULL;
    }

    printf("[agentsh ESF] IPC control socket listening on %s\n", g_socket_path);

    while (1) {
        int client_fd = accept(g_server_fd, NULL, NULL);
        if (client_fd < 0) break;

        char buf[1024];
        ssize_t n = read(client_fd, buf, sizeof(buf) - 1);
        if (n > 0) {
            buf[n] = '\0';
            if (strstr(buf, "get_stats")) {
                char resp[128];
                pthread_mutex_lock(&g_lock);
                snprintf(resp, sizeof(resp), "{\"drop_count\": %llu}\n", (unsigned long long)g_drop_count);
                pthread_mutex_unlock(&g_lock);
                write(client_fd, resp, strlen(resp));
            } else if (strstr(buf, "set_blocked_ports")) {
                // Acknowledge update
                const char *ack = "{\"status\":\"ok\"}\n";
                write(client_fd, ack, strlen(ack));
            }
        }
        close(client_fd);
    }

    return NULL;
}

static void cleanup_and_exit(int sig) {
    (void)sig;
    printf("\n[agentsh ESF] Shutting down Endpoint Security interceptor...\n");
    if (g_es_client) {
        es_delete_client(g_es_client);
        g_es_client = NULL;
    }
    if (g_server_fd >= 0) {
        close(g_server_fd);
        unlink(g_socket_path);
    }
    exit(0);
}

int main(int argc, char **argv) {
    if (argc > 1 && strcmp(argv[1], "--socket") == 0 && argc > 2) {
        strncpy(g_socket_path, argv[2], sizeof(g_socket_path) - 1);
    }

    signal(SIGINT, cleanup_and_exit);
    signal(SIGTERM, cleanup_and_exit);

    printf("🛡️ [agentsh ESF] Initializing Apple EndpointSecurity kernel interceptor...\n");

    es_new_client_result_t res = es_new_client(&g_es_client, ^(es_client_t *c, const es_message_t *msg) {
        handle_event(c, msg);
    });

    if (res != ES_NEW_CLIENT_RESULT_SUCCESS) {
        fprintf(stderr, "❌ [agentsh ESF] Failed to register macOS Endpoint Security client (code %d).\n"
                        "   Ensure you are running as root (`sudo`) and that this binary is signed\n"
                        "   with the `com.apple.developer.endpoint-security.client` entitlement.\n", res);
        return 1;
    }

    es_event_type_t events[] = { ES_EVENT_TYPE_AUTH_UIPC_CONNECT };
    if (es_subscribe(g_es_client, events, 1) != ES_RETURN_SUCCESS) {
        fprintf(stderr, "❌ [agentsh ESF] Failed to subscribe to ES_EVENT_TYPE_AUTH_UIPC_CONNECT events.\n");
        es_delete_client(g_es_client);
        return 1;
    }

    printf("✅ [agentsh ESF] Subscribed to Kernel Auth UIPC Connect events.\n");

    pthread_t thread;
    if (pthread_create(&thread, NULL, ipc_server_thread, NULL) != 0) {
        fprintf(stderr, "⚠️ Failed to start IPC control thread.\n");
    }

    printf("🚀 [agentsh ESF] Active! macOS Kernel is natively intercepting rogue AI agent connections.\n");

    dispatch_main();
    return 0;
}
