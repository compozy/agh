import { cva } from "class-variance-authority";

const progressIndicatorVariants = cva("h-full transition-all", {
  variants: {
    tone: {
      accent: "bg-accent",
      success: "bg-success",
      warning: "bg-warning",
      danger: "bg-danger",
      info: "bg-info",
      neutral: "bg-neutral",
    },
  },
  defaultVariants: {
    tone: "accent",
  },
});

export { progressIndicatorVariants };
