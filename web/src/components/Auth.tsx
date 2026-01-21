import { type FormEvent, useState } from "react";
import { useWsStore } from "../lib/wsStore";

export function Auth() {
	const [url, setUrl] = useState("ws://localhost:8080/ws");
	const [token, setToken] = useState("");
	const [isConnecting, setIsConnecting] = useState(false);
	const [error, setError] = useState<string | null>(null);
	const connect = useWsStore((s) => s.connect);

	const handleSubmit = async (e: FormEvent) => {
		e.preventDefault();
		setIsConnecting(true);
		setError(null);

		try {
			await connect(url, token);
		} catch (err) {
			setError((err as Error).message);
		} finally {
			setIsConnecting(false);
		}
	};

	return (
		<div className="min-h-screen flex items-center justify-center bg-gray-900 px-4">
			<div className="w-full max-w-md">
				<div className="text-center mb-8">
					<h1 className="text-3xl font-bold text-white mb-2">Devport</h1>
					<p className="text-gray-400">Mobile AI Programming Platform</p>
				</div>

				<form
					onSubmit={handleSubmit}
					className="bg-gray-800 rounded-lg p-6 shadow-xl"
				>
					<div className="mb-4">
						<label
							htmlFor="url"
							className="block text-sm font-medium text-gray-300 mb-2"
						>
							Server URL
						</label>
						<input
							id="url"
							type="text"
							value={url}
							onChange={(e) => setUrl(e.target.value)}
							className="w-full px-4 py-3 bg-gray-700 border border-gray-600 rounded-lg text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
							placeholder="ws://localhost:8080/ws"
						/>
					</div>

					<div className="mb-6">
						<label
							htmlFor="token"
							className="block text-sm font-medium text-gray-300 mb-2"
						>
							Access Token
						</label>
						<input
							id="token"
							type="password"
							value={token}
							onChange={(e) => setToken(e.target.value)}
							className="w-full px-4 py-3 bg-gray-700 border border-gray-600 rounded-lg text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
							placeholder="Enter your access token"
						/>
					</div>

					{error && (
						<div className="mb-4 p-3 bg-red-900/50 border border-red-700 rounded-lg text-red-300 text-sm">
							{error}
						</div>
					)}

					<button
						type="submit"
						disabled={isConnecting || !token}
						className="w-full py-3 px-4 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-600 disabled:cursor-not-allowed text-white font-medium rounded-lg transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 focus:ring-offset-gray-800"
					>
						{isConnecting ? "Connecting..." : "Connect"}
					</button>
				</form>
			</div>
		</div>
	);
}
