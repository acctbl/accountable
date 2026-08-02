import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Button } from "./button";

describe("Button", () => {
	it("renders its label", () => {
		const label = "Save";
		render(<Button>{label}</Button>);
		expect(screen.getByRole("button", { name: label })).toBeTruthy();
	});
});
