import { describe, it, expect } from 'vitest';
import { shouldSyncTab, buildTabUrl } from './review-tab-sync';

describe('shouldSyncTab', () => {
	describe('movies tab (default)', () => {
		it('returns false when URL has no ?tab param (regression: prevents infinite loop)', () => {
			expect(shouldSyncTab(null, 'movies')).toBe(false);
		});

		it('returns true when URL has ?tab=failed', () => {
			expect(shouldSyncTab('failed', 'movies')).toBe(true);
		});

		it('returns true when URL has invalid ?tab value', () => {
			expect(shouldSyncTab('garbage', 'movies')).toBe(true);
		});
	});

	describe('failed tab', () => {
		it('returns false when URL has ?tab=failed', () => {
			expect(shouldSyncTab('failed', 'failed')).toBe(false);
		});

		it('returns true when URL has no ?tab param', () => {
			expect(shouldSyncTab(null, 'failed')).toBe(true);
		});

		it('returns true when URL has ?tab=movies', () => {
			expect(shouldSyncTab('movies', 'failed')).toBe(true);
		});
	});
});

describe('buildTabUrl', () => {
	const baseUrl = new URL('http://localhost/review/job-123');

	it('removes ?tab param when activeTab is movies', () => {
		const url = buildTabUrl(baseUrl, 'movies');
		expect(url.searchParams.has('tab')).toBe(false);
		expect(url.toString()).toBe('http://localhost/review/job-123');
	});

	it('sets ?tab=failed when activeTab is failed', () => {
		const url = buildTabUrl(baseUrl, 'failed');
		expect(url.searchParams.get('tab')).toBe('failed');
	});

	it('replaces existing ?tab value', () => {
		const urlWithTab = new URL('http://localhost/review/job-123?tab=movies');
		const result = buildTabUrl(urlWithTab, 'failed');
		expect(result.searchParams.get('tab')).toBe('failed');
	});

	it('preserves unrelated query params', () => {
		const urlWithOtherParams = new URL('http://localhost/review/job-123?foo=bar&baz=qux');
		const result = buildTabUrl(urlWithOtherParams, 'failed');
		expect(result.searchParams.get('foo')).toBe('bar');
		expect(result.searchParams.get('baz')).toBe('qux');
		expect(result.searchParams.get('tab')).toBe('failed');
	});

	it('does not mutate the input URL', () => {
		const url = new URL('http://localhost/review/job-123?tab=movies');
		buildTabUrl(url, 'failed');
		expect(url.searchParams.get('tab')).toBe('movies');
	});
});
