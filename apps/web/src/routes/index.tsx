import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import * as React from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useApi } from "@/lib/api";
import { formatApiError } from "@/lib/api-error";
import { LocaleSwitcher } from "@/components/locale-switcher";

export const Route = createFileRoute("/")({ component: Home });

function Home() {
	const { system } = useApi();
	const intl = useIntl();
	const runtime = useQuery({
		queryKey: ["system", "runtime"],
		queryFn: () => system.getRuntime({}),
		retry: false,
	});

	React.useEffect(() => {
		if (runtime.data || runtime.error) {
			performance.mark("accountable:route-useful-content");
		}
	}, [runtime.data, runtime.error]);

	return (
		<main id="main-content" className="flex min-h-svh text-sm" tabIndex={-1}>
			<div className="min-w-0 space-y-2 p-6">
				<LocaleSwitcher />
				<p>
					<FormattedMessage
						id="home.description"
						defaultMessage="Local transport shell"
					/>
				</p>
				<p role="status" aria-live="polite" data-testid="runtime-status">
					{runtime.isPending ? (
						<FormattedMessage
							id="runtime.loading"
							defaultMessage="Connecting to the local API…"
						/>
					) : null}
					{runtime.data ? (
						<FormattedMessage
							id="runtime.connected"
							defaultMessage="Connected ({releaseId})"
							values={{ releaseId: runtime.data.releaseId }}
						/>
					) : null}
					{runtime.error ? formatApiError(intl, runtime.error) : null}
				</p>
			</div>
		</main>
	);
}
