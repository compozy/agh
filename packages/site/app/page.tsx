import {
  Hero,
  TwoPillars,
  HowItWorks,
  RuntimeFeatures,
  ProtocolSection,
  Architecture,
  Comparison,
  FinalCta,
} from "@/components/landing";

export default function HomePage() {
  return (
    <main>
      <Hero />
      <TwoPillars />
      <HowItWorks />
      <RuntimeFeatures />
      <ProtocolSection />
      <Architecture />
      <Comparison />
      <FinalCta />
    </main>
  );
}
