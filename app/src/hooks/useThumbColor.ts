import { useEffect, useState } from 'react';
import ImageColors from 'react-native-image-colors';
import { COLORS } from '../theme';

/**
 * Extracts a vivid accent colour from a remote thumbnail URL.
 *
 * Priority (Android Palette):
 *   lightVibrant → vibrant → muted → lightMuted → dominant
 *
 * We skip `dominant` first because YouTube thumbnails usually have a dark
 * dominant (letterbox / background), which makes every card look the same.
 * We fall to dominant only as the last resort, and then sanitise it if it's
 * too dark by blending it toward white.
 */
function isTooDark(hex: string): boolean {
    // Parse #RRGGBB and compute relative luminance (simple check).
    const r = parseInt(hex.slice(1, 3), 16);
    const g = parseInt(hex.slice(3, 5), 16);
    const b = parseInt(hex.slice(5, 7), 16);
    // Perceived brightness (0-255)
    const brightness = (r * 299 + g * 587 + b * 114) / 1000;
    return brightness < 80; // below this = nearly black
}

/**
 * Lighten a hex colour towards white by a factor (0-1).
 * Used as a last-resort when only a dark dominant colour is available.
 */
function lighten(hex: string, factor: number = 0.5): string {
    const r = Math.min(255, Math.round(parseInt(hex.slice(1, 3), 16) + (255 - parseInt(hex.slice(1, 3), 16)) * factor));
    const g = Math.min(255, Math.round(parseInt(hex.slice(3, 5), 16) + (255 - parseInt(hex.slice(3, 5), 16)) * factor));
    const b = Math.min(255, Math.round(parseInt(hex.slice(5, 7), 16) + (255 - parseInt(hex.slice(5, 7), 16)) * factor));
    return `#${r.toString(16).padStart(2, '0')}${g.toString(16).padStart(2, '0')}${b.toString(16).padStart(2, '0')}`;
}

/** Fallback palette used when the image returns no usable colour. */
const FALLBACK_PALETTE = [
    '#FBBF24', '#F472B6', '#34D399', '#60A5FA',
    '#A78BFA', '#F97316', '#10B981', '#818CF8',
];

function hashFallback(id: string): string {
    let h = 0;
    for (let i = 0; i < id.length; i++) h = (h * 31 + id.charCodeAt(i)) >>> 0;
    return FALLBACK_PALETTE[h % FALLBACK_PALETTE.length];
}

export function useThumbColor(thumbnailUrl?: string, id?: string): string {
    const [color, setColor] = useState<string>(
        id ? hashFallback(id) : COLORS.primary
    );

    useEffect(() => {
        if (!thumbnailUrl) return;

        ImageColors.getColors(thumbnailUrl, {
            fallback: id ? hashFallback(id) : COLORS.primary,
            cache: true,
            key: `thumb_${id ?? thumbnailUrl}`,
        }).then((result) => {
            let picked: string | undefined;

            if (result.platform === 'android') {
                // Prefer dominant (most displayed colour in the thumbnail)
                picked =
                    result.dominant ||
                    result.vibrant ||
                    result.muted ||
                    result.lightVibrant ||
                    result.lightMuted ||
                    undefined;
            } else if (result.platform === 'ios') {
                picked =
                    result.secondary ||
                    result.primary ||
                    result.background ||
                    result.detail ||
                    undefined;
            } else if (result.platform === 'web') {
                picked = result.vibrant || result.muted || undefined;
            }

            if (picked) {
                // If we ended up with a very dark colour, lighten it so it's
                // still visible as a card shadow / background tint.
                setColor(isTooDark(picked) ? lighten(picked, 0.55) : picked);
            }
        }).catch(() => {
            // Silently keep the hash-based fallback
        });
    }, [thumbnailUrl, id]);

    return color;
}
