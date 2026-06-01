import type { ProgressMessage } from '$lib/api/types';

export const TERMINAL_STATUSES = new Set(['completed', 'failed', 'cancelled', 'organized', 'reverted']);

export function isTerminalStatus(status: string | null | undefined): boolean {
	if (!status) return false;
	return TERMINAL_STATUSES.has(status.toLowerCase());
}

export function computeJobProgress(
	messagesByFile: Record<string, ProgressMessage> | undefined,
	totalFiles: number,
	restProgress: number,
	isRunning: boolean,
	finishedCount: number = 0,
): number {
	if (totalFiles === 0) return restProgress;
	if (!isRunning) {
		return Math.min(Math.round((finishedCount / totalFiles) * 100), 100);
	}
	const entries = Object.values(messagesByFile ?? {});
	let runningProgress = 0;
	for (const m of entries) {
		if (!isTerminalStatus(m.status)) {
			runningProgress += Math.max(0, Math.min(100, m.progress));
		}
	}
	const totalProgress = finishedCount + runningProgress / 100;
	return Math.min(Math.round((totalProgress / totalFiles) * 100), 100);
}
