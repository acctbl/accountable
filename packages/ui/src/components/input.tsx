import { cn } from "@accountable/ui/lib/utils";
import { Input as InputPrimitive } from "@base-ui/react/input";

function Input({ className, dir = "auto", ...props }: InputPrimitive.Props) {
  return (
    <InputPrimitive
      data-slot="input"
      dir={dir}
      className={cn(
        "flex h-7 w-full min-w-0 rounded-md border border-border bg-transparent px-2 py-1 text-base/relaxed xl:text-xs/relaxed transition-[color,box-shadow] outline-none",
        "placeholder:text-muted-foreground",
        "focus-visible:border-ring focus-visible:ring-2 focus-visible:ring-ring/30",
        "disabled:pointer-events-none disabled:opacity-50",
        "aria-invalid:border-destructive aria-invalid:ring-2 aria-invalid:ring-destructive/20",
        className,
      )}
      {...props}
    />
  );
}

export { Input };
