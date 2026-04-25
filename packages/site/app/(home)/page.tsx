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
      <BridgesSection />
      <ExtensibilitySection />
      <InstallSection />
      <Comparison />
      <FinalCta />
    </main>
  );
}
