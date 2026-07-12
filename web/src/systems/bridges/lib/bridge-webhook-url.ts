import type { BridgeDetailResponse } from "../types";

const SPECIAL_IPV4_RANGES = [
  [0x00000000, 0xff000000],
  [0x0a000000, 0xff000000],
  [0x64400000, 0xffc00000],
  [0x7f000000, 0xff000000],
  [0xa9fe0000, 0xffff0000],
  [0xac100000, 0xfff00000],
  [0xc0000000, 0xffffff00],
  [0xc0000200, 0xffffff00],
  [0xc0586300, 0xffffff00],
  [0xc0a80000, 0xffff0000],
  [0xc6120000, 0xfffe0000],
  [0xc6336400, 0xffffff00],
  [0xcb007100, 0xffffff00],
  [0xe0000000, 0xf0000000],
  [0xf0000000, 0xf0000000],
] as const;

const SPECIAL_IPV6_PREFIXES = [
  ["64:ff9b::", 96],
  ["64:ff9b:1::", 48],
  ["100::", 64],
  ["2001::", 32],
  ["2001:2::", 48],
  ["2001:db8::", 32],
  ["2002::", 16],
] as const;

export function readWebhookPublicUrl(
  providerConfig: BridgeDetailResponse["bridge"]["provider_config"]
): string | null {
  if (
    providerConfig === null ||
    typeof providerConfig !== "object" ||
    Array.isArray(providerConfig)
  ) {
    return null;
  }
  const webhook = providerConfig.webhook;
  if (webhook === null || typeof webhook !== "object" || Array.isArray(webhook)) {
    return null;
  }
  if (!("public_url" in webhook)) {
    return null;
  }
  const rawPublicUrl = webhook.public_url;
  if (typeof rawPublicUrl !== "string" || rawPublicUrl.trim() === "") {
    return null;
  }

  const publicUrl = rawPublicUrl.trim();
  try {
    const parsed = new URL(publicUrl);
    const hostname = parsed.hostname.toLowerCase();
    if (
      parsed.protocol !== "https:" ||
      hostname === "" ||
      hostname === "localhost" ||
      hostname.endsWith(".localhost") ||
      parsed.username !== "" ||
      parsed.password !== "" ||
      parsed.hash !== "" ||
      isNonPublicIPLiteral(hostname)
    ) {
      return null;
    }
    return publicUrl;
  } catch {
    return null;
  }
}

function isNonPublicIPLiteral(hostname: string): boolean {
  const bareHostname =
    hostname.startsWith("[") && hostname.endsWith("]") ? hostname.slice(1, -1) : hostname;
  const ipv4 = parseIPv4(bareHostname);
  if (ipv4 !== null) {
    return SPECIAL_IPV4_RANGES.some(([network, mask]) => (ipv4 & mask) === (network & mask));
  }
  if (!bareHostname.includes(":")) {
    return false;
  }

  const ipv6 = parseIPv6(bareHostname);
  if (ipv6 === null) {
    return true;
  }
  const first = ipv6[0];
  const unspecifiedOrLoopback = ipv6.slice(0, 7).every(group => group === 0) && ipv6[7] <= 1;
  const uniqueLocal = (first & 0xfe00) === 0xfc00;
  const linkLocal = (first & 0xffc0) === 0xfe80;
  const multicast = (first & 0xff00) === 0xff00;
  const ipv4Mapped = ipv6.slice(0, 5).every(group => group === 0) && ipv6[5] === 0xffff;
  if (ipv4Mapped) {
    const mapped = (ipv6[6] << 16) | ipv6[7];
    return SPECIAL_IPV4_RANGES.some(([network, mask]) => (mapped & mask) === (network & mask));
  }
  return (
    unspecifiedOrLoopback ||
    uniqueLocal ||
    linkLocal ||
    multicast ||
    SPECIAL_IPV6_PREFIXES.some(([prefix, bits]) => matchesIPv6Prefix(ipv6, prefix, bits))
  );
}

function parseIPv4(value: string): number | null {
  const parts = value.split(".");
  if (parts.length !== 4) return null;
  const octets = parts.map(part => (/^(0|[1-9]\d{0,2})$/.test(part) ? Number(part) : -1));
  if (octets.some(octet => octet < 0 || octet > 255)) return null;
  return octets.reduce((result, octet) => ((result << 8) | octet) >>> 0, 0);
}

function parseIPv6(value: string): number[] | null {
  if (value.includes("%") || value.split("::").length > 2) return null;
  const [leftRaw, rightRaw] = value.split("::");
  const left = parseIPv6Side(leftRaw ?? "");
  const right = parseIPv6Side(rightRaw ?? "");
  if (left === null || right === null) return null;
  const hasCompression = value.includes("::");
  const missing = 8 - left.length - right.length;
  if ((!hasCompression && missing !== 0) || (hasCompression && missing < 1)) return null;
  return [...left, ...Array.from({ length: missing }, () => 0), ...right];
}

function parseIPv6Side(value: string): number[] | null {
  if (value === "") return [];
  const parts = value.split(":");
  const last = parts.at(-1);
  if (last?.includes(".")) {
    const ipv4 = parseIPv4(last);
    if (ipv4 === null) return null;
    parts.splice(
      parts.length - 1,
      1,
      ((ipv4 >>> 16) & 0xffff).toString(16),
      (ipv4 & 0xffff).toString(16)
    );
  }
  if (parts.some(part => !/^[\da-f]{1,4}$/i.test(part))) return null;
  return parts.map(part => Number.parseInt(part, 16));
}

function matchesIPv6Prefix(address: number[], prefix: string, bits: number): boolean {
  const expected = parseIPv6(prefix);
  if (expected === null) return false;
  const fullGroups = Math.floor(bits / 16);
  const remainder = bits % 16;
  for (let index = 0; index < fullGroups; index += 1) {
    if (address[index] !== expected[index]) return false;
  }
  if (remainder === 0) return true;
  const mask = (0xffff << (16 - remainder)) & 0xffff;
  return (address[fullGroups] & mask) === (expected[fullGroups] & mask);
}
