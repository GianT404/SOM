import React, { useEffect, useState, useRef } from 'react';
import {
    View, Text, Image, TouchableOpacity, StyleSheet, Animated, Easing, Dimensions
} from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';
import { BlurView } from 'expo-blur';
import ImageColors from 'react-native-image-colors';
import { COLORS, SPACING, RADIUS, FONT_SIZE, RETRO_BORDER, RETRO_SHADOW_SM } from '../theme';
import { usePlayer } from '../contexts/PlayerContext';
import MarqueeText from './MarqueeText';

// --- Component chính: MiniPlayer ---
const MiniPlayer: React.FC<{ onPress: () => void }> = ({ onPress }) => {
    const { currentTrack, isPlaying, isLoading, pause, resume, position, duration } = usePlayer();
    const [mainColor, setMainColor] = useState(COLORS.secondary);

    // Logic trích xuất màu từ Thumbnail
    useEffect(() => {
        if (currentTrack?.thumbnail) {
            ImageColors.getColors(currentTrack.thumbnail, {
                fallback: COLORS.secondary,
                cache: true,
                key: currentTrack.id,
            }).then((colors) => {
                if (colors.platform === 'android') {
                    setMainColor(colors.vibrant || colors.dominant);
                } else if (colors.platform === 'ios') {
                    setMainColor(colors.primary || colors.background);
                }
            });
        }
    }, [currentTrack?.thumbnail]);

    if (!currentTrack) return null;

    const progress = duration > 0 ? position / duration : 0;

    return (
        <TouchableOpacity
            style={[styles.container, { backgroundColor: mainColor }, RETRO_SHADOW_SM]}
            onPress={onPress}
            activeOpacity={0.95}
        >
            {/* Lớp mờ (Glassmorphism) */}
            <BlurView intensity={20} tint="dark" style={StyleSheet.absoluteFill} />

            {/* Thanh Progress */}
            <View style={styles.progressBar}>
                <View style={[styles.progressFill, { width: `${progress * 100}%` }]} />
            </View>

            <View style={styles.content}>
                <Image source={{ uri: currentTrack.thumbnail }} style={styles.thumb} />

                <View style={styles.info}>
                    <MarqueeText
                        text={currentTrack.title}
                        style={styles.title}
                    />
                    <Text style={styles.artist} numberOfLines={1}>
                        {currentTrack.uploader}
                    </Text>
                </View>

                <TouchableOpacity
                    style={[styles.playBtn, { backgroundColor: COLORS.background }]}
                    onPress={(e) => {
                        e.stopPropagation();
                        isPlaying ? pause() : resume();
                    }}
                >
                    <MaterialIcons
                        name={isLoading ? 'hourglass-empty' : (isPlaying ? 'pause' : 'play-arrow')}
                        size={24}
                        color={COLORS.textDark}
                    />
                </TouchableOpacity>
            </View>
        </TouchableOpacity>
    );
};

const styles = StyleSheet.create({
    container: {
        ...RETRO_BORDER,
        borderRadius: 20,
        marginHorizontal: SPACING.md,
        overflow: 'hidden',
        height: 75,
        justifyContent: 'center',
    },
    progressBar: {
        position: 'absolute',
        top: 0,
        left: 0,
        right: 0,
        height: 4,
        backgroundColor: 'rgba(255,255,255,0.2)',
    },
    progressFill: {
        height: '100%',
        backgroundColor: COLORS.primary,
    },
    content: {
        flexDirection: 'row',
        alignItems: 'center',
        paddingHorizontal: SPACING.md,
    },
    thumb: {
        width: 48,
        height: 48,
        borderRadius: 12,
        ...RETRO_BORDER,
    },
    info: {
        flex: 1,
        marginLeft: SPACING.md,
        justifyContent: 'center',
    },
    marqueeContainer: {
        overflow: 'hidden',
        width: '100%',
    },
    title: {
        color: COLORS.background,
        fontSize: 16,
        fontWeight: '800',
        textShadowColor: 'rgba(0, 0, 0, 0.75)',
        textShadowOffset: { width: -1, height: 1 },
        textShadowRadius: 10,
        flexWrap: 'nowrap',
    },
    artist: {
        color: 'rgba(255,255,255,0.8)',
        fontSize: 12,
        fontWeight: '600',
        marginTop: 2,
    },
    playBtn: {
        width: 40,
        height: 40,
        borderRadius: 20,
        ...RETRO_BORDER,
        justifyContent: 'center',
        alignItems: 'center',
    },
});

export default MiniPlayer;