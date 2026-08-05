import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import * as React from "react";
import { FormattedMessage, useIntl } from "react-intl";
import { useApi } from "@/lib/api";
import { formatApiError } from "@/lib/api-error";
import { LocaleSwitcher } from "@/components/locale-switcher";
import { Button } from "@accountable/ui/components/button";
import { PlusIcon } from "@phosphor-icons/react";

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
		<main id="main-content" className="flex min-h-svh" tabIndex={-1}>
			<div className="min-w-0 space-y-2 p-6">
				<h1 className="sr-only">
					<FormattedMessage id="home.title" defaultMessage="Accountable" />
				</h1>
				<div className="flex items-center">
					<LocaleSwitcher />
					<Button size="icon" variant="ghost">
						<PlusIcon />
					</Button>
				</div>
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
