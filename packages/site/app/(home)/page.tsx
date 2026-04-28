import {
  BentoSection,
  BridgesSection,
  Comparison,
  ExtensibilitySection,
  FeaturesSection,
  FinalCta,
  Hero,
  InstallSection,
  NetworkSection,
  RuntimeSection,
  SandboxSection,
  SupportedAgents,
} from "@/components/landing";

export default function HomePage() {
  return (
    <main className="site-home">
      <Hero />
      <BentoSection />
      <FeaturesSection />
      <SupportedAgents />
      <NetworkSection />
      <RuntimeSection />
      <SandboxSection />
      <BridgesSection />
      <ExtensibilitySection />
      <InstallSection />
      <Comparison />
      <FinalCta />
    </main>
  );
}
