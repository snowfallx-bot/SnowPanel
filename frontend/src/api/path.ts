export function withEncodedSegment(prefix: string, segment: string, suffix = "") {
  return `${prefix}/${encodeURIComponent(segment)}${suffix}`;
}
