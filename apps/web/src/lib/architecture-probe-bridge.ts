import {
	ArchitectureProbeService,
	FailureKind,
} from "@accountable/proto/accountable/probe/v1/architecture_probe_pb";
import { SystemService } from "@accountable/proto/accountable/system/v1/system_pb";
import { create } from "@bufbuild/protobuf";
import { DurationSchema } from "@bufbuild/protobuf/wkt";
import { createClient } from "@connectrpc/connect";
import { createApiTransport } from "./api";
import { parseApiError } from "./api-error";
import type { RuntimeConfig } from "./runtime-config";

type FailureName = "invalid_input" | "unavailable";

type StreamState = {
	done: boolean;
	error: string | null;
	sequences: number[];
};

function milliseconds(value: number) {
	return create(DurationSchema, {
		seconds: BigInt(Math.floor(value / 1000)),
		nanos: (value % 1000) * 1_000_000,
	});
}

export function installArchitectureProbeBridge(config: RuntimeConfig): void {
	const transport = createApiTransport(config);
	const probe = createClient(ArchitectureProbeService, transport);
	const system = createClient(SystemService, transport);
	let streamState: StreamState = { done: false, error: null, sequences: [] };

	const bridge = {
		async getRuntime() {
			const response = await system.getRuntime({});
			return {
				releaseId: response.releaseId,
				requestId: response.requestId,
			};
		},
		async fail(name: FailureName) {
			const kind =
				name === "invalid_input"
					? FailureKind.INVALID_INPUT
					: FailureKind.UNAVAILABLE;
			try {
				await probe.fail({ kind });
				throw new Error("architecture failure probe unexpectedly succeeded");
			} catch (error) {
				const parsed = parseApiError(error);
				return {
					code: parsed.code,
					category: parsed.category,
					messageKey: parsed.messageKey,
					problemId: parsed.problemId,
					requestId: parsed.requestId,
					fieldViolations: parsed.fieldViolations.map((violation) => ({
						fieldPath: violation.fieldPath,
						code: violation.code,
					})),
				};
			}
		},
		async cancel() {
			const controller = new AbortController();
			const request = probe.wait(
				{ delay: milliseconds(10_000) },
				{ signal: controller.signal },
			);
			setTimeout(() => controller.abort(), 25);
			try {
				await request;
				throw new Error(
					"architecture cancellation probe unexpectedly completed",
				);
			} catch (error) {
				return parseApiError(error).code;
			}
		},
		async correlate() {
			const response = await probe.correlate({});
			return { requestId: response.requestId, traceId: response.traceId };
		},
		async cookieRoundTrip() {
			await probe.cookieRoundTrip({ setCookie: true });
			const response = await probe.cookieRoundTrip({ setCookie: false });
			return response.cookieReceived;
		},
		startStream() {
			streamState = { done: false, error: null, sequences: [] };
			return new Promise<StreamState>((resolveFirstDelivery) => {
				void (async () => {
					let delivered = false;
					try {
						for await (const message of probe.stream({
							count: 3,
							interval: milliseconds(120),
						})) {
							streamState = {
								...streamState,
								sequences: [...streamState.sequences, message.sequence],
							};
							if (!delivered) {
								delivered = true;
								resolveFirstDelivery({
									...streamState,
									sequences: [...streamState.sequences],
								});
							}
						}
						streamState = { ...streamState, done: true };
						if (!delivered) resolveFirstDelivery({ ...streamState });
					} catch (error) {
						streamState = {
							...streamState,
							done: true,
							error: error instanceof Error ? error.message : "stream failed",
						};
						if (!delivered) resolveFirstDelivery({ ...streamState });
					}
				})();
			});
		},
		streamState() {
			return {
				...streamState,
				sequences: [...streamState.sequences],
			};
		},
	};

	Object.defineProperty(window, "__accountableArchitectureProbe", {
		configurable: true,
		value: Object.freeze(bridge),
	});
}

declare global {
	interface Window {
		__accountableArchitectureProbe?: {
			cancel(): Promise<number>;
			cookieRoundTrip(): Promise<boolean>;
			correlate(): Promise<{ requestId: string; traceId: string }>;
			getRuntime(): Promise<{ releaseId: string; requestId: string }>;
			fail(name: FailureName): Promise<{
				code: number;
				category: string | null;
				messageKey: string | null;
				problemId: string | null;
				requestId: string | null;
				fieldViolations: Array<{ fieldPath: string; code: string }>;
			}>;
			startStream(): Promise<StreamState>;
			streamState(): StreamState;
		};
	}
}
