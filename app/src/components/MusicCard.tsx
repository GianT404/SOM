import React, { useEffect, useState } from 'react';
import { View, Text, Image, TouchableOpacity, StyleSheet } from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';
import ImageColors from 'react-native-image-colors';
import { COLORS, SPACING, NEO_BRUTALISM, RADIUS } from '../theme';

// Định nghĩa type cho ImageColors response
interface ImageColorsResult {
    platform: 'android' | 'web' | 'ios';
    vibrant?: string;
    dominant?: string;
    primary?: string;
    background?: string;
}

interface MusicCardProps {
    id: string;
    title: string;
    thumbnail: string;
    duration: number;
    uploader: string;
    showDeleteBtn?: boolean;
    onPress: () => void;
    onLongPress?: () => void;
    onDeletePress?: () => void;
}

const MusicCard: React.FC<MusicCardProps & { index?: number }> = ({
    title, thumbnail, duration, onPress, onLongPress, onDeletePress, showDeleteBtn, id, uploader
}) => {
    const [accentColor, setAccentColor] = useState(COLORS.primary);

    // Trích xuất màu từ thumbnail để làm bóng đổ khối (Neo-Brutalism shadow)
    // Ưu tiên màu rực rỡ (vibrant) để UI nổi bật, fallback sang màu dominant (chiếm diện tích)
    // Cache được bật để lưu kết quả tính toán, không tốn CPU lần tải tiếp theo
    useEffect(() => {
        if (thumbnail) {
            ImageColors.getColors(thumbnail, {
                fallback: COLORS.primary,
                cache: true,  // Lưu cache để tránh tính toán lại CPU
                key: id,
            }).then((colors: ImageColorsResult) => {
                // Android & Web: Ưu tiên vibrant (màu rực rỡ), fallback sang dominant (màu chiếm diện tích)
                if (colors.platform === 'android' || colors.platform === 'web') {
                    setAccentColor(colors.dominant || colors.vibrant || COLORS.primary);
                }
                // iOS: Sử dụng primary hoặc background
                else if (colors.platform === 'ios') {
                    setAccentColor(colors.primary || colors.background || COLORS.primary);
                }
                // Fallback cho platform khác
                else {
                    setAccentColor(colors.dominant || colors.vibrant || COLORS.primary);
                }
            }).catch(() => {
                // Nếu lỗi xảy ra, dùng màu default
                setAccentColor(COLORS.primary);
            });
        }
    }, [thumbnail, id]);

    const formatDuration = (seconds: number) => {
        const m = Math.floor(seconds / 60).toString().padStart(2, '0');
        const s = Math.floor(seconds % 60).toString().padStart(2, '0');
        return `${m}:${s}`;
    };

    return (
        <View style={styles.container}>
            <View style={[NEO_BRUTALISM.shadowSm, { backgroundColor: accentColor }]} />

            {/* Thân Card chính */}
            <TouchableOpacity
                style={styles.card}
                onPress={onPress}
                onLongPress={onLongPress}
                activeOpacity={0.9}
            >
                {/* Ảnh Thumbnail */}
                <View style={styles.imageWrapper}>
                    <Image source={{ uri: thumbnail }} style={styles.image} resizeMode="cover" />
                </View>

                <View style={styles.content}>
                    <Text style={styles.duration}>{formatDuration(duration)}</Text>

                    {/* Phần Title */}
                    <View style={styles.titleContainer}>
                        <Text style={styles.title} numberOfLines={1}>{title}</Text>
                    </View>

                    {/* Uploader (giữ nguyên logic) */}
                    <Text style={styles.uploader} numberOfLines={1}>{uploader}</Text>
                </View>

                {/* Nút Xóa */}
                {showDeleteBtn && (
                    <TouchableOpacity
                        style={styles.deleteBtn}
                        onPress={onDeletePress}
                        activeOpacity={0.8}
                    >
                        <MaterialIcons name="close" size={14} color="#FFF" />
                    </TouchableOpacity>
                )}
            </TouchableOpacity>
        </View>
    );
};

const styles = StyleSheet.create({
    container: {
        width: '47%',
        marginBottom: SPACING.lg,
        position: 'relative',
    },

    card: {
        width: '100%',
        backgroundColor: COLORS.card,
        borderRadius: RADIUS.sm,
        borderWidth: 2,
        borderColor: '#1A1A1A',
        padding: 8,
        overflow: 'hidden',
    },
    imageWrapper: {
        width: '100%',
        height: 110,
        borderRadius: 15,
        borderWidth: 2,
        borderColor: '#1A1A1A',
        overflow: 'hidden',
    },
    image: {
        width: '100%',
        height: '100%'
    },
    content: {
        marginTop: 8,
        paddingBottom: 2
    },
    duration: {
        fontSize: 11,
        fontWeight: '900',
        color: COLORS.textMuted,
        marginBottom: 2
    },
    titleContainer: {
        marginBottom: 2,
    },
    title: {
        fontSize: 14,
        fontWeight: '900',
        color: COLORS.textDark,
    },
    uploader: {
        fontSize: 10,
        color: COLORS.textMuted,
        fontWeight: '600',
    },
    deleteBtn: {
        position: 'absolute',
        top: 6,
        right: 6,
        width: 22,
        height: 22,
        borderRadius: 11,
        backgroundColor: COLORS.red,
        justifyContent: 'center',
        alignItems: 'center',
        zIndex: 10,
        borderWidth: 1.5,
        borderColor: '#1A1A1A',
    }
});

export default MusicCard;