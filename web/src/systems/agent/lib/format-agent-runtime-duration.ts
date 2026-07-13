/** Format summed session elapsed seconds for Overview/Sessions metric display. */
export function formatAgentRuntimeDuration(totalSeconds: number): string | null {
  if (!Number.isFinite(totalSeconds) || totalSeconds <= 0) return "0s";
  const total = Math.floor(totalSeconds);
  if (total < 60) return `${total}s`;
  const minutes = Math.floor(total / 60);
  if (minutes < 60) return `${minutes}m`;
  const hours = Math.floor(minutes / 60);
  const remainderMinutes = minutes % 60;
  return remainderMinutes === 0 ? `${hours}h` : `${hours}h ${remainderMinutes}m`;
}
