import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { ThemeProvider } from "./theme-provider";

const originalStorage = Object.getOwnPropertyDescriptor(window, "localStorage");

function replaceLocalStorage(descriptor: PropertyDescriptor) {
	Object.defineProperty(window, "localStorage", {
		configurable: true,
		...descriptor,
	});
}

afterEach(() => {
	cleanup();
	if (originalStorage) {
		Object.defineProperty(window, "localStorage", originalStorage);
	} else {
		Reflect.deleteProperty(window, "localStorage");
	}
	document.documentElement.classList.remove("light", "dark");
});

describe("theme provider", () => {
	it("renders children when storage access is denied", () => {
		replaceLocalStorage({
			get() {
				throw new DOMException("denied", "SecurityError");
			},
		});

		render(
			<ThemeProvider defaultTheme="light" storageKey="acctbl-theme">
				<p data-testid="shell-content" />
			</ThemeProvider>,
		);

		expect(screen.getByTestId("shell-content")).toBeDefined();
		expect(document.documentElement.classList.contains("light")).toBe(true);
	});

	it("cycles the theme with the hotkey when storage writes are refused", () => {
		replaceLocalStorage({
			value: {
				getItem: () => null,
				setItem: () => {
					throw new DOMException("quota", "QuotaExceededError");
				},
			},
		});

		render(
			<ThemeProvider defaultTheme="light" storageKey="acctbl-theme">
				<p data-testid="shell-content" />
			</ThemeProvider>,
		);

		fireEvent.keyDown(window, { key: "d" });

		expect(document.documentElement.classList.contains("dark")).toBe(true);
	});
});
