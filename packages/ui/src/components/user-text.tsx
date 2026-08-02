import type * as React from "react";

function UserText({ children, ...props }: React.ComponentProps<"bdi">) {
	return (
		<bdi dir="auto" {...props}>
			{children}
		</bdi>
	);
}

export { UserText };
