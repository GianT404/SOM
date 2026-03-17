import React, { useState, useEffect, useRef } from 'react';
import {
    View, Text, ScrollView, Image, TouchableOpacity, StyleSheet, StatusBar, ActivityIndicator,
} from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';
import { COLORS, SPACING, RADIUS, FONT_SIZE, RETRO_BORDER, RETRO_SHADOW } from '../theme';
import { usePlayer } from '../contexts/PlayerContext';
import api, { LyricLine, LyricsData } from '../services/api';

const formatTime = (ms: number) => {
    const s = Math.floor(ms / 1000);
    return `${Math.floor(s / 60).toString().padStart(2, '0')}:${(s % 60).toString().padStart(2, '0')}`;
};

const LyricsScreen: React.FC<{ navigation: any }> = ({ navigation }) => {
    const { currentTrack, position, isPlaying, pause, resume, duration } = usePlayer();
    const [lyricsData, setLyricsData] = useState<LyricsData[]>([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');
    const [selectedLanguage, setSelectedLanguage] = useState<string>('vi');
    const currentSec = position / 1000;
    const progress = duration > 0 ? position / duration : 0;

    // Cache lyrics mỗi bài để tránh re-fetch khi chuyển bài
    const lyricsCache = useRef<{ [trackId: string]: LyricsData[] }>({});
    const trackIdRef = useRef<string>('');

    useEffect(() => {
        if (!currentTrack) return;

        // Cập nhật track ID ref để tránh race condition
        trackIdRef.current = currentTrack.id;

        setLoading(true);
        setError('');

        // Ưu tiên 1: Lyrics đã tải về từ offline cache (từ playlist)
        if (currentTrack.lyrics && Array.isArray(currentTrack.lyrics) && currentTrack.lyrics.length > 0) {
            // Nếu là offline track (có localUri), không bao giờ gọi API
            const data = [{ 
                language: currentTrack.lyrics[0].language === undefined ? 'vi' : currentTrack.lyrics[0].language,
                lines: currentTrack.lyrics 
            }];
            if (currentTrack.lyrics[0].language === undefined) {
                // Nếu không có language field, wrap vào array
                setLyricsData([{ language: 'vi', lines: currentTrack.lyrics }]);
            } else if (Array.isArray(currentTrack.lyrics) && currentTrack.lyrics[0].lines) {
                // Nếu đã là array LyricsData, dùng trực tiếp
                setLyricsData(currentTrack.lyrics);
            } else {
                setLyricsData([{ language: 'vi', lines: currentTrack.lyrics }]);
            }
            lyricsCache.current[currentTrack.id] = data;
            setLoading(false);
            return;
        }

        // Ưu tiên 2: Kiểm tra cache đã fetch trước đó
        if (lyricsCache.current[currentTrack.id]) {
            setLyricsData(lyricsCache.current[currentTrack.id]);
            setLoading(false);
            return;
        }

        // Ưu tiên 3: Fetch từ API (chỉ khi không phải offline track)
        const currentTrackId = currentTrack.id;
        
        api.getLyrics(currentTrack.id)
            .then(data => {
                // Chỉ update nếu vẫn là track này (tránh bug race condition)
                if (trackIdRef.current === currentTrackId) {
                    if (data && data.length > 0) {
                        setLyricsData(data);
                        lyricsCache.current[currentTrackId] = data;
                    } else {
                        setLyricsData([]);
                    }
                }
            })
            .catch((e: Error) => {
                // Chỉ update error nếu vẫn là track này
                if (trackIdRef.current === currentTrackId) {
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
                }
            })
            .finally(() => {
                // Chỉ update loading nếu vẫn là track này
                if (trackIdRef.current === currentTrackId) {
                    setLoading(false);
                }
            });
    }, [currentTrack?.id]);

    if (!currentTrack) {
        return (
            <View style={[styles.container, styles.center]}>
                <Text style={styles.emptyText}>No track playing</Text>
            </View>
        );
    }

    // Lấy lyrics của ngôn ngữ được chọn hoặc mặc định
    let activeLyrics: LyricLine[] = [];
    if (lyricsData.length > 0) {
        const match = lyricsData.find(d => d.language === selectedLanguage);
        activeLyrics = match ? match.lines : lyricsData[0].lines;
    }

    const activeIndex = activeLyrics.findIndex(
        (line, i) => currentSec >= line.start && (i === activeLyrics.length - 1 || currentSec < activeLyrics[i + 1].start)
    );

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
                <>
                    {/* Language Selector */}
                    {lyricsData.length > 1 && (
                        <View style={styles.languageSelector}>
                            {lyricsData.map((data) => (
                                <TouchableOpacity
                                    key={data.language}
                                    style={[styles.langBtn, selectedLanguage === data.language && styles.langBtnActive]}
                                    onPress={() => setSelectedLanguage(data.language)}
                                >
                                    <Text style={[styles.langBtnText, selectedLanguage === data.language && styles.langBtnTextActive]}>
                                        {data.language.toUpperCase()}
                                    </Text>
                                </TouchableOpacity>
                            ))}
                        </View>
                    )}
                    <ScrollView style={styles.lyricsScroll} contentContainerStyle={styles.lyricsContent}>
                        {activeLyrics.length > 0 ? (
                            activeLyrics.map((line, index) => (
                                <Text key={index} style={[styles.lyricLine, index === activeIndex && styles.lyricLineActive]}>
                                    {line.text}
                                </Text>
                            ))
                        ) : (
                            <View style={styles.center}>
                                <MaterialIcons name="subtitles-off" size={48} color={COLORS.textMuted} />
                                <Text style={styles.emptyText}>No lyrics available</Text>
                            </View>
                        )}
                        <View style={{ height: 200 }} />
                    </ScrollView>
                </>
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
    languageSelector: {
        flexDirection: 'row', gap: SPACING.sm, paddingHorizontal: SPACING.lg, paddingVertical: SPACING.md,
        justifyContent: 'center', alignItems: 'center',
    },
    langBtn: {
        paddingHorizontal: SPACING.md, paddingVertical: SPACING.xs, backgroundColor: COLORS.card,
        ...RETRO_BORDER, borderRadius: RADIUS.md,
    },
    langBtnActive: {
        backgroundColor: COLORS.primary,
    },
    langBtnText: { fontSize: FONT_SIZE.xs, fontWeight: '600', color: COLORS.textMuted },
    langBtnTextActive: { color: COLORS.textLight },
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
