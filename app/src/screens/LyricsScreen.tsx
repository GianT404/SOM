import React, { useState, useEffect, useRef } from 'react';
import {
    View, Text, ScrollView, Image, TouchableOpacity, StyleSheet, StatusBar, ActivityIndicator,
} from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';
import { COLORS, SPACING, RADIUS, FONT_SIZE, RETRO_BORDER, RETRO_SHADOW } from '../theme';
import { usePlayer } from '../contexts/PlayerContext';
import api, { LyricLine } from '../services/api';

const formatTime = (ms: number) => {
    const s = Math.floor(ms / 1000);
    return `${Math.floor(s / 60).toString().padStart(2, '0')}:${(s % 60).toString().padStart(2, '0')}`;
};

const LyricsScreen: React.FC<{ navigation: any }> = ({ navigation }) => {
    const { currentTrack, position, isPlaying, pause, resume, duration } = usePlayer();
    const [lyrics, setLyrics] = useState<LyricLine[]>([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');
    const currentSec = position / 1000;
    const progress = duration > 0 ? position / duration : 0;

    useEffect(() => {
        if (!currentTrack) return;
        setLoading(true);
        setError('');
        api.getLyrics(currentTrack.id)
            .then(data => {
                if (data && data.length > 0) {
                    setLyrics(data[0].lines || []);
                } else {
                    setLyrics([]);
                }
            })
            .catch((e: Error) => {
                let msg = 'No lyrics available';
                if (e.message) {
                    if (e.message.includes('HTTP Error 429') || e.message.includes('Too Many Requests')) {
                        msg = 'YouTube Rate Limit Exceeded.\nPlease wait a while and try again later.';
                    } else if (e.message.includes('no subtitles available')) {
                        msg = 'No subtitles available for this track.';
                    } else {
                        msg = e.message;
                    }
                }
                setError(msg);
            })
            .finally(() => setLoading(false));
    }, [currentTrack?.id]);

    const activeIndex = lyrics.findIndex(
        (line, i) => currentSec >= line.start && (i === lyrics.length - 1 || currentSec < lyrics[i + 1].start)
    );

    if (!currentTrack) {
        return (
            <View style={[styles.container, styles.center]}>
                <Text style={styles.emptyText}>No track playing</Text>
            </View>
        );
    }

    return (
        <View style={styles.container}>
            <StatusBar barStyle="dark-content" backgroundColor={COLORS.background} />
            {/* Header */}
            <View style={styles.header}>
                <TouchableOpacity style={[styles.backBtn, RETRO_SHADOW]} onPress={() => navigation.goBack()}>
                    <MaterialIcons name="arrow-back" size={22} color={COLORS.textDark} />
                </TouchableOpacity>
                <Text style={styles.headerTitle}>Lyrics</Text>
                <TouchableOpacity>
                    <MaterialIcons name="share" size={22} color={COLORS.textDark} />
                </TouchableOpacity>
            </View>
            {/* Track Card */}
            <View style={[styles.trackCard, RETRO_SHADOW]}>
                <Image source={{ uri: currentTrack.thumbnail }} style={styles.trackArt} />
                <View style={styles.trackMeta}>
                    <Text style={styles.trackTitle} numberOfLines={1}>{currentTrack.title}</Text>
                    <Text style={styles.trackArtist}>{currentTrack.uploader}</Text>
                </View>
            </View>
            {/* Lyrics */}
            {loading ? (
                <View style={styles.center}><ActivityIndicator size="large" color={COLORS.primary} /></View>
            ) : error ? (
                <View style={styles.center}>
                    <MaterialIcons name="subtitles-off" size={48} color={COLORS.textMuted} />
                    <Text style={styles.emptyText}>{error}</Text>
                </View>
            ) : (
                <ScrollView style={styles.lyricsScroll} contentContainerStyle={styles.lyricsContent}>
                    {lyrics.map((line, index) => (
                        <Text key={index} style={[styles.lyricLine, index === activeIndex && styles.lyricLineActive]}>
                            {line.text}
                        </Text>
                    ))}
                    <View style={{ height: 200 }} />
                </ScrollView>
            )}

            {/* Bottom Player Controls */}
            <View style={styles.bottomPlayer}>
                <View style={[styles.progressBox, RETRO_SHADOW]}>
                    <Text style={styles.timeText}>{formatTime(position)}</Text>
                    <View style={styles.progressTrack}>
                        <View style={[styles.progressFill, { width: `${progress * 100}%` }]} />
                    </View>
                    <Text style={styles.timeText}>{formatTime(duration)}</Text>
                </View>
                <View style={styles.controlRow}>
                    <TouchableOpacity><MaterialIcons name="skip-previous" size={32} color={COLORS.textDark} /></TouchableOpacity>
                    <TouchableOpacity style={[styles.playBtn, RETRO_SHADOW]} onPress={isPlaying ? pause : resume}>
                        <MaterialIcons name={isPlaying ? 'pause' : 'play-arrow'} size={36} color={COLORS.textLight} />
                    </TouchableOpacity>
                    <TouchableOpacity><MaterialIcons name="skip-next" size={32} color={COLORS.textDark} /></TouchableOpacity>
                </View>
            </View>
        </View>
    );
};

const styles = StyleSheet.create({
    container: { flex: 1, backgroundColor: COLORS.background },
    center: { flex: 1, justifyContent: 'center', alignItems: 'center' },
    emptyText: { color: COLORS.textMuted, fontSize: FONT_SIZE.md, marginTop: SPACING.md },
    header: {
        flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center',
        paddingHorizontal: SPACING.lg, paddingTop: SPACING.xxl, paddingBottom: SPACING.md,
    },
    backBtn: {
        width: 36, height: 36, borderRadius: 18, backgroundColor: COLORS.primary,
        ...RETRO_BORDER, justifyContent: 'center', alignItems: 'center',
    },
    headerTitle: { fontSize: FONT_SIZE.lg, fontWeight: '700', color: COLORS.textDark },
    trackCard: {
        flexDirection: 'row', alignItems: 'center', marginHorizontal: SPACING.lg,
        backgroundColor: COLORS.card, ...RETRO_BORDER, borderRadius: RADIUS.full,
        padding: SPACING.sm, paddingRight: SPACING.md,
    },
    trackArt: { width: 44, height: 44, borderRadius: 22, ...RETRO_BORDER },
    trackMeta: { flex: 1, marginLeft: SPACING.md },
    trackTitle: { fontSize: FONT_SIZE.md, fontWeight: '700', color: COLORS.textDark },
    trackArtist: { fontSize: FONT_SIZE.xs, color: COLORS.textMuted, fontWeight: '500' },
    lyricsScroll: { flex: 1 },
    lyricsContent: { paddingHorizontal: SPACING.xl, paddingTop: SPACING.lg },
    lyricLine: { fontSize: FONT_SIZE.lg, fontWeight: '500', color: COLORS.textMuted, lineHeight: 32, marginBottom: SPACING.md },
    lyricLineActive: { color: COLORS.textDark, fontWeight: '900', fontSize: FONT_SIZE.xxl, lineHeight: 38 },
    bottomPlayer: { paddingHorizontal: SPACING.lg, paddingBottom: SPACING.xl },
    progressBox: {
        flexDirection: 'row', alignItems: 'center', backgroundColor: COLORS.card,
        ...RETRO_BORDER, borderRadius: RADIUS.full, padding: SPACING.md, gap: SPACING.sm,
    },
    progressTrack: { flex: 1, height: 4, backgroundColor: COLORS.background, borderRadius: 2 },
    progressFill: { height: '100%', backgroundColor: COLORS.primary, borderRadius: 2 },
    timeText: { fontSize: FONT_SIZE.xs, color: COLORS.textMuted, fontWeight: '600', minWidth: 36, textAlign: 'center' },
    controlRow: { flexDirection: 'row', justifyContent: 'center', alignItems: 'center', marginTop: SPACING.md, gap: SPACING.xl },
    playBtn: {
        width: 60, height: 60, borderRadius: 30, backgroundColor: COLORS.secondary,
        ...RETRO_BORDER, justifyContent: 'center', alignItems: 'center',
    },
});

export default LyricsScreen;
