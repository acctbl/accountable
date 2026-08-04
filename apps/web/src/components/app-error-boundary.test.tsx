import { cleanup, render, screen } from "@testing-library/react";
import { IntlProvider } from "react-intl";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AppErrorBoundary } from "./app-error-boundary";

function ThrowingChild(): never {
	throw new Error("render failed");
}

afterEach(() => {
	cleanup();
	vi.restoreAllMocks();
});

describe("app error boundary", () => {
	it("renders the reload fallback when a child render throws", () => {
		vi.spyOn(console, "error").mockImplementation(() => {});

		render(
			<IntlProvider locale="en" defaultLocale="en" onError={() => {}}>
				<AppErrorBoundary>
					<ThrowingChild />
				</AppErrorBoundary>
			</IntlProvider>,
		);

		expect(screen.getByRole("alert")).toBeDefined();
		expect(screen.getByRole("button", { name: "Reload" })).toBeDefined();
	});
});
