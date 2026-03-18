import { useEffect, useState } from 'react';
import ImageColors from 'react-native-image-colors';
import { COLORS } from '../theme';

/**
 * Định nghĩa type cho ImageColors response
 * Hỗ trợ tất cả các platform (Android, iOS, Web)
 */
interface ImageColorsResult {
    platform: 'android' | 'ios' | 'web';
    vibrant?: string;
    lightVibrant?: string;
    darkVibrant?: string;
    muted?: string;
    lightMuted?: string;
    darkMuted?: string;
    dominant?: string;
    detail?: string;
    secondary?: string;
    primary?: string;
    background?: string;
}

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
    '#7e7171',
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
        }).then((result: ImageColorsResult) => {
            // 1. Lập danh sách ứng viên theo thứ tự ưu tiên chuẩn xác nhất cho app âm nhạc
            let candidates: (string | undefined)[] = [];

            if (result.platform === 'android') {
                // Tôn trọng triết lý: Rực rỡ (Vibrant) lên ngôi, màu chủ đạo (Dominant - dễ bị dính viền đen) xuống bét.
                candidates = [
                    result.dominant,
                    result.lightVibrant,
                    result.muted,
                    result.vibrant
                ];
            } else if (result.platform === 'ios') {
                // iOS: Detail thường là màu nhấn rực rỡ, Secondary là màu phụ, Primary hay bị dính màu nền.
                candidates = [
                    result.detail,
                    result.secondary,
                    result.primary,
                    result.background
                ];
            } else if (result.platform === 'web') {
                candidates = [result.vibrant, result.dominant];
            }

            // 2. Lọc bỏ các giá trị undefined để lấy mảng màu thực tế
            const validColors = candidates.filter((c): c is string => !!c);

            // 3. Vòng gửi xe: Tìm màu ĐẦU TIÊN trong danh sách ứng viên mà KHÔNG BỊ QUÁ TỐI
            let picked = validColors.find((c) => !isTooDark(c));

            // 4. Phương án dự phòng (Fallback)
            if (picked) {
                // Đã tìm được màu xịn
                setColor(picked);
            } else if (validColors.length > 0) {
                // Trường hợp bi đát: Cả cái thumbnail toàn màu tối thui (ví dụ video nhạc buồn, cover màu đen).
                // Ta đành lấy thằng rực rỡ nhất (đứng đầu mảng) và nâng độ sáng của nó lên một chút cho dịu mắt.
                setColor(lighten(validColors[0], 0.45)); 
            }

        }).catch(() => {
            // Lỗi mạng hoặc không lấy được, cứ để im xài hashFallback đã cấu hình ở useState
        });
    }, [thumbnailUrl, id]);

    return color;
}
