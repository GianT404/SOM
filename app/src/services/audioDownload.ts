import * as FileSystem from 'expo-file-system/legacy';

export interface AudioDownloadProgress {
    totalBytesWritten: number;
    totalBytesExpectedToWrite: number;
}

type ProgressCallback = (progress: AudioDownloadProgress) => void;

const activeDownloads = new Map<string, Promise<string>>();

/** Download a complete audio response to a temporary file before publishing it to the cache. */
export async function downloadAudioAtomically(
    url: string,
    destinationUri: string,
    headers: Record<string, string>,
    onProgress?: ProgressCallback,
): Promise<string> {
    const active = activeDownloads.get(destinationUri);
    if (active) return active;

    const task = downloadAndCommit(url, destinationUri, headers, onProgress);
    activeDownloads.set(destinationUri, task);
    try {
        return await task;
    } finally {
        if (activeDownloads.get(destinationUri) === task) {
            activeDownloads.delete(destinationUri);
        }
    }
}

async function downloadAndCommit(
    url: string,
    destinationUri: string,
    headers: Record<string, string>,
    onProgress?: ProgressCallback,
): Promise<string> {
    const partialUri = `${destinationUri}.part`;
    try {
        let result: { uri: string; status: number; headers?: Record<string, string> } | undefined;
        let retriedRateLimit = false;
        while (true) {
            await FileSystem.deleteAsync(partialUri, { idempotent: true });
            const download = FileSystem.createDownloadResumable(
                url,
                partialUri,
                { headers },
                progress => onProgress?.({
                    totalBytesWritten: progress.totalBytesWritten,
                    totalBytesExpectedToWrite: progress.totalBytesExpectedToWrite,
                }),
            );
            result = await download.downloadAsync();
            if (!result?.uri) {
                throw new Error('Audio download returned no file');
            }
            if (result.status === 429 && !retriedRateLimit) {
                const delayMs = retryAfterMs(result.headers || {});
                await FileSystem.deleteAsync(partialUri, { idempotent: true });
                retriedRateLimit = true;
                await new Promise<void>(resolve => setTimeout(resolve, delayMs));
                continue;
            }
            break;
        }
        if (result.status !== 200) {
            throw new Error(`Audio request failed with HTTP ${result.status}`);
        }

        const info = await FileSystem.getInfoAsync(partialUri);
        if (!info.exists || typeof info.size !== 'number' || info.size < 4) {
            throw new Error('Audio download is empty or incomplete');
        }

        const contentLength = Object.entries(result.headers || {}).find(
            ([name]) => name.toLowerCase() === 'content-length',
        )?.[1];
        const expectedSize = Number(contentLength);
        if (Number.isFinite(expectedSize) && expectedSize > 0 && info.size !== expectedSize) {
            throw new Error(`Audio download is incomplete (${info.size}/${expectedSize} bytes)`);
        }

        await FileSystem.deleteAsync(destinationUri, { idempotent: true });
        await FileSystem.moveAsync({ from: partialUri, to: destinationUri });
        return destinationUri;
    } catch (error) {
        await FileSystem.deleteAsync(partialUri, { idempotent: true }).catch(() => undefined);
        throw error;
    }
}

function retryAfterMs(headers: Record<string, string>): number {
    const raw = Object.entries(headers).find(
        ([name]) => name.toLowerCase() === 'retry-after',
    )?.[1];
    const seconds = Number(raw);
    if (Number.isFinite(seconds) && seconds > 0) {
        return Math.min(seconds, 30) * 1000;
    }

    const retryAt = raw ? Date.parse(raw) : NaN;
    if (Number.isFinite(retryAt)) {
        return Math.min(Math.max(retryAt - Date.now(), 1000), 30000);
    }
    return 1000;
}
