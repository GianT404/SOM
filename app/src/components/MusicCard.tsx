import React, { useEffect, useState } from 'react';
import { View, Text, Image, TouchableOpacity, StyleSheet } from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';
import ImageColors from 'react-native-image-colors';
import { COLORS, SPACING, RETRO_BORDER } from '../theme';

interface MusicCardProps {
    id: string;
    title: string;
    thumbnail: string;
    duration: number;
    uploader: string;
    onPress: () => void;
}

const MusicCard: React.FC<MusicCardProps & { index?: number }> = ({ title, thumbnail, duration, onPress, id }) => {
    const [accentColor, setAccentColor] = useState(COLORS.primary);

    // Trích xuất màu từ thumbnail để làm bóng đổ nội bộ
    useEffect(() => {
        if (thumbnail) {
            ImageColors.getColors(thumbnail, {
                fallback: COLORS.primary,
                cache: true,
                key: id,
            }).then((colors) => {
                if (colors.platform === 'android') {
                    setAccentColor(colors.vibrant || colors.dominant || COLORS.primary);
                }
            });
        }
    }, [thumbnail, id]);

    const formatDuration = (seconds: number) => {
        const m = Math.floor(seconds / 60).toString().padStart(2, '0');
        const s = Math.floor(seconds % 60).toString().padStart(2, '0');
        return `${m}:${s}`;
    };

    return (
        <TouchableOpacity
            style={styles.card}
            onPress={onPress}
            activeOpacity={0.9}
        >
            {/* Ảnh Thumbnail */}
            <View style={styles.imageWrapper}>
                <Image source={{ uri: thumbnail }} style={styles.image} resizeMode="cover" />
            </View>

            <View style={styles.content}>
                <Text style={styles.duration}>{formatDuration(duration)}</Text>

                {/* Phần Title  */}
                <View style={styles.titleContainer}>
                    <Text style={styles.title} numberOfLines={1}>{title}</Text>
                </View>
            </View>
            <View style={[styles.bottomBar, { backgroundColor: accentColor }]} />
        </TouchableOpacity>
    );
};

const styles = StyleSheet.create({
    card: {
        width: '47%',
        backgroundColor: COLORS.card,
        borderRadius: 20,
        ...RETRO_BORDER,
        marginBottom: SPACING.lg,
        padding: 10,
        overflow: 'hidden',
        position: 'relative',
    },
    imageWrapper: {
        width: '100%',
        height: 120,
        borderRadius: 15,
        ...RETRO_BORDER,
        overflow: 'hidden',
    },
    image: { width: '100%', height: '100%' },
    content: { marginTop: 10, paddingBottom: 5 },
    duration: { fontSize: 12, fontWeight: '900', color: COLORS.textMuted, marginBottom: 4 },
    titleContainer: {
        position: 'relative',
        marginBottom: 3,
    },
    title: {
        fontSize: 14,
        fontWeight: '900',
        color: COLORS.textDark,
        zIndex: 2,
    },
    titleShadow: {
        position: 'absolute',
        bottom: -2,
        left: 0,
        width: '60%',
        height: 6,
        opacity: 0.6,
        zIndex: 1,
    },

    footer: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginTop: 5,
    },
    uploader: { fontSize: 10, color: COLORS.textMuted, flex: 1 },
    playIconBox: {
        width: 28,
        height: 28,
        borderRadius: 14,
        backgroundColor: COLORS.textDark,
        justifyContent: 'center',
        alignItems: 'center',
    },
    bottomBar: {
        position: 'absolute',
        bottom: 0,
        left: 0,
        right: 0,
        height: 5,
    }
});

export default MusicCard;