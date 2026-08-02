import { Button } from "@accountable/ui/components/button";
import { createFileRoute } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { LocaleSwitcher } from "@/components/locale-switcher";
import { systemClient } from "@/lib/api";
import { formatApiError } from "@/lib/api-error";
import { PlusIcon } from "@phosphor-icons/react";

export const Route = createFileRoute("/")({ component: Home });

function Home() {
	const intl = useIntl();
	const [runtime, setRuntime] = useState<{
		releaseId: string;
		requestId: string;
	} | null>(null);
	const [error, setError] = useState<unknown>(null);

	useEffect(() => {
		let cancelled = false;

		void systemClient
			.getRuntime({})
			.then((res) => {
				if (!cancelled) {
					setRuntime({
						releaseId: res.releaseId,
						requestId: res.requestId,
					});
					setError(null);
				}
			})
			.catch((err: unknown) => {
				if (!cancelled) {
					setError(err);
					setRuntime(null);
				}
			});

		return () => {
			cancelled = true;
		};
	}, []);

	return (
		<main className="flex min-h-svh p-6">
			<div className="flex max-w-md min-w-0 flex-col gap-4 text-sm leading-loose">
				<div>
					<h1 className="font-medium">
						<FormattedMessage id="home.title" defaultMessage="Project ready!" />
					</h1>

					<div className="mt-2 flex items-center gap-2">
						<Button>
							<PlusIcon />
							<FormattedMessage id="home.buttonLabel" defaultMessage="Button" />
						</Button>
						<LocaleSwitcher />
					</div>
				</div>
				<div className="font-mono text-xs text-muted-foreground">
					{error ? (
						<p role="alert">{formatApiError(intl, error)}</p>
					) : runtime ? (
						<p title={runtime.requestId}>
							<FormattedMessage
								id="home.runtimeConnected"
								defaultMessage="Connected ({releaseId})"
								values={{ releaseId: runtime.releaseId }}
							/>
						</p>
					) : (
						<p>
							<FormattedMessage
								id="home.runtimeLoading"
								defaultMessage="Connecting…"
							/>
						</p>
					)}
				</div>
				<div className="font-mono text-xs text-muted-foreground">
					<FormattedMessage
						id="home.themeHint"
						defaultMessage="(Press <key>d</key> to cycle dark / light / system)"
						values={{
							key: (chunks) => <kbd>{chunks}</kbd>,
						}}
					/>
				</div>
			</div>
		</main>
	);
}
