import { cn } from "@accountable/ui/lib/utils";
import type * as React from "react";

function DirectionalIcon({
	className,
	children,
	...props
}: React.ComponentProps<"span">) {
	return (
		<span
			data-slot="directional-icon"
			className={cn("inline-flex rtl:-scale-x-100", className)}
			{...props}
		>
			{children}
		</span>
	);
}

export { DirectionalIcon };
