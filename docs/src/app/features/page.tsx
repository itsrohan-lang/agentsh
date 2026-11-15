"use client"
import Navbar from '@/components/Navbar';

export default function FeaturesPage() {
  return (
    <div className="min-h-screen flex flex-col font-[family-name:var(--font-sans)] relative overflow-hidden bg-[#05050a] text-zinc-300">
      <div className="fixed inset-0 z-0 bg-[linear-gradient(to_right,#ffffff05_1px,transparent_1px),linear-gradient(to_bottom,#ffffff05_1px,transparent_1px)] bg-[size:4rem_4rem]"></div>
      
      <Navbar />

      <main className="flex flex-col items-center w-full z-10 pt-16 px-8 pb-32 max-w-6xl mx-auto">
        <div className="text-center mb-16">
          <h1 className="text-5xl font-black text-white mb-6 tracking-tight">Unprecedented Agent Safety.</h1>
          <p className="text-zinc-400 text-xl max-w-2xl mx-auto">
            Stop relying on prompt engineering for safety. Rely on container isolation, git plumbing, and zero-trust egress proxying.
          </p>
        </div>
        
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 w-full">
          <div className="glass-card p-8 flex flex-col gap-4 hover:border-[var(--accent)] transition-all duration-300 group">
            <div className="text-4xl group-hover:scale-110 transition-transform origin-left">🛡️</div>
            <h3 className="text-xl font-bold text-white">Container Sandboxing</h3>
            <p className="text-sm text-zinc-400 leading-relaxed">Runs agents inside a capability-dropped Docker container (--cap-drop=ALL) with strict cgroup memory limits (2GB) and CPU limits (1.0 core).</p>
          </div>
          
          <div className="glass-card p-8 flex flex-col gap-4 hover:border-[#8a2be2] transition-all duration-300 group">
            <div className="text-4xl group-hover:scale-110 transition-transform origin-left">⏪</div>
            <h3 className="text-xl font-bold text-white">Zero-Copy Time Travel</h3>
            <p className="text-sm text-zinc-400 leading-relaxed">agentsh takes zero-overhead snapshots of your working directory using Git&apos;s low-level plumbing objects. If an AI ruins your code, `agentsh rewind` instantly restores it.</p>
          </div>
          
          <div className="glass-card p-8 flex flex-col gap-4 hover:border-[var(--accent)] transition-all duration-300 group">
            <div className="text-4xl group-hover:scale-110 transition-transform origin-left">🔑</div>
            <h3 className="text-xl font-bold text-white">Zero-Trust Key Proxy</h3>
            <p className="text-sm text-zinc-400 leading-relaxed">Agents run with dummy API keys inside the container. The host ZeroTrustProxy intercepts egress requests and substitutes real keys so agents never hold real secrets.</p>
          </div>

          <div className="glass-card p-8 flex flex-col gap-4 hover:border-[#8a2be2] transition-all duration-300 group">
            <div className="text-4xl group-hover:scale-110 transition-transform origin-left">🔍</div>
            <h3 className="text-xl font-bold text-white">Indirect Prompt Guard</h3>
            <p className="text-sm text-zinc-400 leading-relaxed">Inspects prompt strings and context files for hidden HTML comments, role-play hijacks, and system prompt override instructions with automatic sanitization.</p>
          </div>

          <div className="glass-card p-8 flex flex-col gap-4 hover:border-[var(--accent)] transition-all duration-300 group">
            <div className="text-4xl group-hover:scale-110 transition-transform origin-left">🔐</div>
            <h3 className="text-xl font-bold text-white">Data Loss Prevention (DLP)</h3>
            <p className="text-sm text-zinc-400 leading-relaxed">Scans file outputs and egress payloads for sensitive credentials (AWS keys, GitHub tokens, generic secrets) and PII with automated redaction.</p>
          </div>

          <div className="glass-card p-8 flex flex-col gap-4 hover:border-[#8a2be2] transition-all duration-300 group">
            <div className="text-4xl group-hover:scale-110 transition-transform origin-left">📊</div>
            <h3 className="text-xl font-bold text-white">Tamper-Evident Ledger</h3>
            <p className="text-sm text-zinc-400 leading-relaxed">Cryptographic HMAC-SHA256 chained audit logs verified against an internal SQLite database and monitored through the Mission Control dashboard.</p>
          </div>
        </div>
      </main>

      <footer className="border-t border-zinc-800 w-full py-12 flex flex-col items-center justify-center text-zinc-500 text-sm z-10 bg-[#05050a] mt-auto">
        <div className="text-lg font-black text-white mb-2">agent<span className="text-[#00f0ff]">sh</span></div>
        <p>A restricted container shell for AI coding agents. © 2026</p>
      </footer>
    </div>
  );
}
