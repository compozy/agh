import {
  Hero,
  TwoPillars,
  HowItWorks,
  ProtocolSection,
  Comparison,
  FinalCta,
} from "@/components/landing";

export default function HomePage() {
  return (
    <main className="site-home">
      <Hero />
      <TwoPillars />
      <ProtocolSection />
      <HowItWorks />
      <Comparison />
      <FinalCta />
    </main>
  );
}
