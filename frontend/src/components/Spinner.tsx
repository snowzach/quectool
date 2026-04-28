// Static class map — Tailwind's JIT only sees full class names in source,
// so `h-${size}` would compile to nothing and the spinner would have 0 size.
const SIZES: Record<string, string> = {
  sm: "h-4 w-4 border-2",
  md: "h-6 w-6 border-2",
  lg: "h-10 w-10 border-[3px]",
  xl: "h-16 w-16 border-4",
};

export function Spinner({ size = "md" }: { size?: keyof typeof SIZES }) {
  return (
    <span className={`inline-block ${SIZES[size]} border-slate-300 border-t-accent rounded-full animate-spin`} />
  );
}
