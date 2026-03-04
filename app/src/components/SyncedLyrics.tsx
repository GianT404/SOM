import React, { useState, useEffect, useRef } from 'react';
import { View, Text, ScrollView, ActivityIndicator, StyleSheet, Dimensions } from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';
import { COLORS, SPACING, FONT_SIZE } from '../theme';
import api, { LyricLine, LyricsData } from '../services/api';

const { width: SCREEN_WIDTH } = Dimensions.get('window');

interface Props {
    track: any;
    position: number; // in milliseconds
    selectedLanguage: string; // The language chosen by the user
    onLanguagesLoaded?: (languages: string[]) => void;
}

const SyncedLyrics: React.FC<Props> = ({ track, position, selectedLanguage, onLanguagesLoaded }) => {
    const [lyricsData, setLyricsData] = useState<LyricsData[]>([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');

    const [scrollViewHeight, setScrollViewHeight] = useState<number>(0);
    const lineLayouts = useRef<{ [key: number]: { y: number; height: number } }>({});

    const scrollViewRef = useRef<ScrollView>(null);
    const currentSec = position / 1000 + 0.5; // 0.8s ahead of audio

    useEffect(() => {
        if (!track) return;

        const handleData = (data: LyricsData[]) => {
            setLyricsData(data);
            if (onLanguagesLoaded) {
                onLanguagesLoaded(data.map(d => d.language));
            }
        };

        if (track.lyrics && Array.isArray(track.lyrics) && track.lyrics.length > 0) {
            if (track.lyrics[0].language === undefined) {
                handleData([{ language: 'vi', lines: track.lyrics }]);
            } else {
                handleData(track.lyrics);
            }
            return;
        }

        setLoading(true);
        setError('');
        api.getLyrics(track.id, {
            title: track.title,
            artist: track.uploader,
            duration: track.duration, // seconds
        })
            .then(data => handleData(data || []))
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
    }, [track]);

    let activeLyrics: LyricLine[] = [];
    if (lyricsData.length > 0) {
        const match = lyricsData.find(d => d.language === selectedLanguage);
        activeLyrics = match ? match.lines : lyricsData[0].lines;
    }

    const activeIndex = activeLyrics.findIndex(
        (line, i) => currentSec >= line.start && (i === activeLyrics.length - 1 || currentSec < activeLyrics[i + 1].start)
    );

    // Xử lý scroll mượt mà và canh giữa hoàn hảo
    useEffect(() => {
        if (activeIndex >= 0 && scrollViewRef.current && scrollViewHeight > 0) {
            const layout = lineLayouts.current[activeIndex];
            if (layout) {
                const centerOffsetY = layout.y - (scrollViewHeight / 2) + (layout.height / 2);
                scrollViewRef.current.scrollTo({
                    y: Math.max(0, centerOffsetY),
                    animated: true
                });
            }
        }
    }, [activeIndex, scrollViewHeight]);

    if (loading) {
        return <View style={styles.center}><ActivityIndicator size="large" color={COLORS.primary} /></View>;
    }

    if (error || lyricsData.length === 0) {
        return (
            <View style={styles.center}>
                <MaterialIcons name="subtitles-off" size={48} color={COLORS.textMuted} />
                <Text style={styles.emptyText}>{error || 'No lyrics available'}</Text>
            </View>
        );
    }

    return (
        <ScrollView
            ref={scrollViewRef}
            style={styles.lyricsScroll}
            onLayout={(e) => setScrollViewHeight(e.nativeEvent.layout.height)}
            contentContainerStyle={[
                styles.lyricsContent,
                scrollViewHeight > 0 && {
                    paddingTop: scrollViewHeight / 2,
                    paddingBottom: scrollViewHeight / 2,
                }
            ]}
            showsVerticalScrollIndicator={false}
        >
            {activeLyrics.map((line, index) => (
                <Text
                    key={index}
                    onLayout={(e) => {
                        lineLayouts.current[index] = {
                            y: e.nativeEvent.layout.y,
                            height: e.nativeEvent.layout.height
                        };
                    }}
                    style={[styles.lyricLine, index === activeIndex && styles.lyricLineActive]}
                >
                    {line.text}
                </Text>
            ))}
        </ScrollView>
    );
};

const styles = StyleSheet.create({
    center: { flex: 1, justifyContent: 'center', alignItems: 'center', width: '100%' },
    emptyText: { color: COLORS.textMuted, fontSize: FONT_SIZE.md, marginTop: SPACING.md },
    lyricsScroll: { flex: 1, width: '100%' },
    lyricsContent: { alignItems: 'center', paddingHorizontal: SPACING.md },
    lyricLine: { fontSize: FONT_SIZE.md, fontWeight: '600', color: COLORS.textMuted, lineHeight: 28, marginBottom: SPACING.sm, textAlign: 'center' },
    lyricLineActive: { color: COLORS.textDark, fontWeight: '900', fontSize: FONT_SIZE.xl, lineHeight: 34, textAlign: 'center' },
});

export default SyncedLyrics;