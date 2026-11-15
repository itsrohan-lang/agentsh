CC ?= clang
CFLAGS ?= -O2 -Wall -Wextra
FRAMEWORKS = -framework EndpointSecurity -framework Foundation -lbsm
TARGET = bin/agentsh-esf

all: $(TARGET)

$(TARGET): src/esf_interceptor.c
	mkdir -p bin
	$(CC) $(CFLAGS) $(FRAMEWORKS) -o $(TARGET) src/esf_interceptor.c

sign: $(TARGET)
	codesign --force --sign - --entitlements esf.entitlements $(TARGET)

clean:
	rm -rf bin
