import React from 'react';
import { View, Text, Image, TouchableOpacity, StyleSheet } from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';
import { COLORS, SPACING, RADIUS, FONT_SIZE, RETRO_BORDER, RETRO_SHADOW_SM } from '../theme';

interface SongRowProps {
    title: string;
    subtitle: string;
    thumbnail: string;
    duration?: number;
    isPlaying?: boolean;
    onPress: () => void;
}

const formatDuration = (seconds: number) => {
    const m = Math.floor(seconds / 60).toString().padStart(2, '0');
    const s = Math.floor(seconds % 60).toString().padStart(2, '0');
    return `${m}:${s}`;
};

const SongRow: React.FC<SongRowProps> = ({ title, subtitle, thumbnail, duration, isPlaying, onPress }) => (
    <TouchableOpacity
        style={[styles.row, RETRO_SHADOW_SM, isPlaying && styles.rowPlaying]}
        onPress={onPress}
        activeOpacity={0.8}
    >
        <Image source={{ uri: thumbnail }} style={styles.thumb} />
        <View style={styles.info}>
            <Text style={styles.title} numberOfLines={1}>{title}</Text>
            {duration !== undefined && (
                <Text style={styles.duration}>{formatDuration(duration)}</Text>
            )}
        </View>
        <TouchableOpacity style={[styles.playBtn, isPlaying && styles.playBtnActive]} onPress={onPress}>
            <MaterialIcons name={isPlaying ? 'pause' : 'play-arrow'} size={22} color={isPlaying ? COLORS.textLight : COLORS.textDark} />
        </TouchableOpacity>
    </TouchableOpacity>
);

const styles = StyleSheet.create({
    row: {
        flexDirection: 'row',
        alignItems: 'center',
        backgroundColor: COLORS.card,
        ...RETRO_BORDER,
        borderRadius: RADIUS.xl,
        padding: SPACING.md,
        marginBottom: SPACING.md,
    },
    rowPlaying: {
        backgroundColor: COLORS.primary + '30',
    },
    thumb: {
        width: 52,
        height: 52,
        borderRadius: RADIUS.md,
        ...RETRO_BORDER,
        backgroundColor: COLORS.secondary + '30',
    },
    info: {
        flex: 1,
        marginLeft: SPACING.md,
    },
    title: {
        fontSize: FONT_SIZE.lg,
        fontWeight: '700',
        color: COLORS.textDark,
    },
    duration: {
        color: COLORS.textMuted,
        fontSize: FONT_SIZE.sm,
        marginTop: 2,
    },
    playBtn: {
        width: 40,
        height: 40,
        borderRadius: 20,
        backgroundColor: COLORS.card,
        ...RETRO_BORDER,
        justifyContent: 'center',
        alignItems: 'center',
    },
    playBtnActive: {
        backgroundColor: COLORS.primary,
    },
});

export default SongRow;
