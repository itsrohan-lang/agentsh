"use client"
import Navbar from '@/components/Navbar';

export default function CommandsPage() {
  return (
    <div className="min-h-screen flex flex-col font-[family-name:var(--font-sans)] relative overflow-hidden bg-[#05050a] text-zinc-300">
      <div className="fixed inset-0 z-0 bg-[linear-gradient(to_right,#ffffff05_1px,transparent_1px),linear-gradient(to_bottom,#ffffff05_1px,transparent_1px)] bg-[size:4rem_4rem]"></div>
      
      <Navbar />

      <main className="flex flex-col items-center w-full z-10 pt-16 px-8 pb-32 max-w-5xl mx-auto">
        <div className="text-center mb-16">
          <h1 className="text-5xl font-black text-white mb-6 tracking-tight">Command Line Interface</h1>
          <p className="text-zinc-400 text-xl max-w-2xl mx-auto">
            A powerful, intuitive toolkit for autonomous agent containment, zero-trust proxying, and instant rollback.
          </p>
        </div>
        
        <div className="flex flex-col gap-6 w-full">
          {[
            { cmd: 'agentsh run <agent>', desc: 'Runs an AI agent (claude, cursor, aider, sh) inside an isolated Docker container with dropped capabilities, resource ceilings, and zero-trust key substitution.' },
            { cmd: 'agentsh rewind', desc: 'Restores the workspace to the pre-run snapshot using zero-copy git plumbing, removing all untracked files created by the agent.' },
            { cmd: 'agentsh checkpoint [save|restore|list]', desc: 'Creates and manages named zero-copy workspace checkpoints for safe experimentation.' },
            { cmd: 'agentsh guard prompt <file|string>', desc: 'Analyzes prompt strings or markdown files for indirect prompt injection vectors and system override attempts.' },
            { cmd: 'agentsh dlp scan <file|string>', desc: 'Scans for sensitive credentials (AWS, GitHub tokens, generic API keys) and PII with automated redaction.' },
            { cmd: 'agentsh mock [generate|serve]', desc: 'Generates synthetic offline API and database mock templates and boots an offline mock server on port 9090.' },
            { cmd: 'agentsh verify', desc: 'Automated CI/CD security gate checking workspace compliance against zero-trust policy.' },
            { cmd: 'agentsh swarm <agents...>', desc: 'Orchestrates multi-agent teams inside an isolated Docker bridge network (StuntNet).' },
            { cmd: 'agentsh init', desc: 'Writes an .agentsh.yaml configuration file in the current directory.' },
          ].map((item, i) => (
            <div key={i} className="glass-card p-8 flex flex-col lg:flex-row items-start lg:items-center justify-between border-zinc-800/50 hover:bg-zinc-900/50 transition">
              <code className="text-[#00f0ff] font-mono text-lg font-bold bg-[#00f0ff]/10 px-4 py-2 rounded-lg mb-4 lg:mb-0">{item.cmd}</code>
              <p className="text-zinc-400 text-base lg:text-right max-w-xl leading-relaxed">{item.desc}</p>
            </div>
          ))}
        </div>
      </main>

      <footer className="border-t border-zinc-800 w-full py-12 flex flex-col items-center justify-center text-zinc-500 text-sm z-10 bg-[#05050a] mt-auto">
        <div className="text-lg font-black text-white mb-2">agent<span className="text-[#00f0ff]">sh</span></div>
        <p>A restricted container shell for AI coding agents. © 2026</p>
      </footer>
    </div>
  );
}
