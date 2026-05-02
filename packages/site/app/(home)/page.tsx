import {
  AutonomyKernelSection,
  BentoSection,
  BridgesSection,
  Comparison,
  ExtensibilitySection,
  FeaturesSection,
  FinalCta,
  Hero,
  InstallSection,
  MemoryDreamSection,
  NetworkSection,
  SupportedAgents,
} from "@/components/landing";

export default function HomePage() {
  return (
    <main id="main-content" className="site-home">
      <Hero />
      <SupportedAgents />
      <NetworkSection />
      <BentoSection />
      <MemoryDreamSection />
      <AutonomyKernelSection />
      <FeaturesSection />
      <ExtensibilitySection />
      <BridgesSection />
      <InstallSection />
      <Comparison />
      <FinalCta />
    </main>
  );
}
