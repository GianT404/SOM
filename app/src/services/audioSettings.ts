import AsyncStorage from '@react-native-async-storage/async-storage';

const AUDIO_SETTINGS_KEY = 'dm4a_audio_settings';

export type BufferSize = 'stable' | 'balanced' | 'fast';
export type SampleRate = 'auto' | '44100' | '48000' | '96000';

export interface AudioSettings {
    bufferSize: BufferSize;
    sampleRate: SampleRate;
}

export const BUFFER_SIZE_OPTIONS: { key: BufferSize; label: string; samples: number }[] = [
    { key: 'stable', label: 'Stable', samples: 2048 },
    { key: 'balanced', label: 'Balanced', samples: 1024 },
    { key: 'fast', label: 'Fast', samples: 512 },
];

export const SAMPLE_RATE_OPTIONS: { key: SampleRate; label: string }[] = [
    { key: 'auto', label: 'Auto' },
    { key: '44100', label: '44.1 kHz' },
    { key: '48000', label: '48 kHz' },
    { key: '96000', label: '96 kHz' },
];

const DEFAULT_SETTINGS: AudioSettings = {
    bufferSize: 'balanced',
    sampleRate: 'auto',
};

export const getAudioSettings = async (): Promise<AudioSettings> => {
    try {
        const json = await AsyncStorage.getItem(AUDIO_SETTINGS_KEY);
        if (json) {
            return { ...DEFAULT_SETTINGS, ...JSON.parse(json) };
        }
        return DEFAULT_SETTINGS;
    } catch {
        return DEFAULT_SETTINGS;
    }
};

export const saveAudioSettings = async (settings: AudioSettings): Promise<void> => {
    await AsyncStorage.setItem(AUDIO_SETTINGS_KEY, JSON.stringify(settings));
};

/**
 * Convert buffer size key to actual sample values for audio configuration.
 */
export const getBufferSamples = (size: BufferSize): number => {
    return BUFFER_SIZE_OPTIONS.find(o => o.key === size)?.samples ?? 1024;
};

/**
 * Convert sample rate key to Hz value (0 = auto/system default).
 */
export const getSampleRateHz = (rate: SampleRate): number => {
    if (rate === 'auto') return 0;
    return parseInt(rate, 10);
};
