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
import { WebSiteJsonLd } from "@/components/seo/structured-data";

export default function HomePage() {
  return (
    <>
      <WebSiteJsonLd />
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
    </>
  );
}
