import { useWsStore } from "../../lib/wsStore";

export function PermissionDialog() {
	const pendingPermission = useWsStore((s) => s.pendingPermission);
	const respondToPermission = useWsStore((s) => s.respondToPermission);

	if (!pendingPermission) return null;

	return (
		<div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
			<div className="bg-gray-800 rounded-xl shadow-2xl max-w-md w-full p-6 border border-gray-700">
				<div className="flex items-center gap-3 mb-4">
					<div className="w-10 h-10 bg-yellow-500/20 rounded-full flex items-center justify-center">
						<svg
							className="w-6 h-6 text-yellow-400"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
							aria-hidden="true"
						>
							<title>Permission</title>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
							/>
						</svg>
					</div>
					<div>
						<h3 className="text-lg font-semibold text-white">
							Permission Request
						</h3>
						<p className="text-sm text-gray-400">
							{pendingPermission.toolName}
						</p>
					</div>
				</div>

				<div className="bg-gray-900 rounded-lg p-4 mb-6">
					<p className="text-gray-300 text-sm whitespace-pre-wrap">
						{pendingPermission.description}
					</p>
				</div>

				<div className="flex gap-3">
					<button
						type="button"
						onClick={() => respondToPermission(false)}
						className="flex-1 px-4 py-3 bg-gray-700 hover:bg-gray-600 text-white rounded-lg transition-colors focus:outline-none focus:ring-2 focus:ring-gray-500"
					>
						Deny
					</button>
					<button
						type="button"
						onClick={() => respondToPermission(true)}
						className="flex-1 px-4 py-3 bg-green-600 hover:bg-green-700 text-white rounded-lg transition-colors focus:outline-none focus:ring-2 focus:ring-green-500"
					>
						Allow
					</button>
				</div>
			</div>
		</div>
	);
}
