export type ReviewTabId = 'movies' | 'failed';

export function shouldSyncTab(currentParam: string | null, activeTab: ReviewTabId): boolean {
	const expectedParam = activeTab === 'movies' ? null : activeTab;
	return currentParam !== expectedParam;
}

export function buildTabUrl(baseUrl: URL, activeTab: ReviewTabId): URL {
	const url = new URL(baseUrl);
	if (activeTab === 'movies') {
		url.searchParams.delete('tab');
	} else {
		url.searchParams.set('tab', activeTab);
	}
	return url;
}
