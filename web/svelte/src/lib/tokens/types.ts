export interface Tokens {
  font: {
    display: string;
    body: string;
    prose: string;
    mono: string;
  };
  color: {
    bg: string;
    bgElev: string;
    bgOverlay: string;
    border: string;
    borderStrong: string;
    text: string;
    textSecondary: string;
    textMuted: string;
    textFaint: string;
    accent: string;
    accentFade: string;
    accentGlow: string;
    statusUp: string;
    statusWarn: string;
    statusDown: string;
  };
  radius: {
    sm: number;
    base: number;
    lg: number;
  };
  motion: {
    fast: number;
    base: number;
    ease: string;
  };
  tracking: {
    label: number;
    display: number;
  };
  shadow: {
    sm: string;
    base: string;
  };
}
