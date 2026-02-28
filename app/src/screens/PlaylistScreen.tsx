import React, { useCallback, useState, useEffect } from 'react';
import {
    View, Text, FlatList, TouchableOpacity, StyleSheet, StatusBar, Alert, Image,
} from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';
import { COLORS, SPACING, RADIUS, FONT_SIZE, RETRO_BORDER, RETRO_SHADOW, RETRO_SHADOW_SM } from '../theme';
import { usePlayer } from '../contexts/PlayerContext';
import { getPlaylist, removeFromPlaylist, OfflineTrack } from '../services/playlistStore';
import { useFocusEffect } from '@react-navigation/native';

const formatDuration = (sec: number) => {
    const m = Math.floor(sec / 60).toString().padStart(2, '0');
    const s = Math.floor(sec % 60).toString().padStart(2, '0');
    return `${m}:${s}`;
};

const PlaylistScreen: React.FC<{ navigation: any }> = ({ navigation }) => {
    const { play, currentTrack } = usePlayer();
    const [tracks, setTracks] = useState<OfflineTrack[]>([]);

    // Reload playlist every time screen is focused
    useFocusEffect(
        useCallback(() => {
            getPlaylist().then(setTracks);
        }, [])
    );

    const handlePlay = useCallback((track: OfflineTrack) => {
        // Play from local file for offline playback
        play({
            id: track.id,
            title: track.title,
            uploader: track.uploader,
            thumbnail: track.thumbnail,
            duration: track.duration,
            localUri: track.localUri,  // <-- local path for offline
            lyrics: track.lyrics,      // <-- pass cached lyrics
        });
        navigation.navigate('NowPlaying');
    }, [play, navigation]);

    const handleDelete = useCallback((track: OfflineTrack) => {
        Alert.alert(
            'Remove Download',
            `Delete "${track.title}" from offline playlist?`,
            [
                { text: 'Cancel', style: 'cancel' },
                {
                    text: 'Delete', style: 'destructive',
                    onPress: async () => {
                        await removeFromPlaylist(track.id);
                        setTracks(prev => prev.filter(t => t.id !== track.id));
                    },
                },
            ]
        );
    }, []);

    return (
        <View style={styles.container}>
            <StatusBar barStyle="dark-content" backgroundColor={COLORS.background} />

            {/* Header */}
            <View style={styles.header}>
                <TouchableOpacity style={[styles.backBtn, RETRO_SHADOW]}>
                    <MaterialIcons name="arrow-back" size={22} color={COLORS.textDark} />
                </TouchableOpacity>
                <Text style={styles.headerTitle}>My Playlist</Text>
                <TouchableOpacity>
                    <MaterialIcons name="share" size={22} color={COLORS.textDark} />
                </TouchableOpacity>
            </View>

            {/* Track Count */}
            <Text style={styles.trackCount}>{tracks.length} downloaded tracks</Text>

            {tracks.length === 0 ? (
                <View style={styles.emptyContainer}>
                    <MaterialIcons name="cloud-download" size={64} color={COLORS.textMuted} />
                    <Text style={styles.emptyTitle}>No offline tracks yet</Text>
                    <Text style={styles.emptySubtitle}>
                        Download songs from the Now Playing screen{'\n'}to listen offline
                    </Text>
                </View>
            ) : (
                <FlatList
                    data={tracks}
                    keyExtractor={(item) => item.id}
                    contentContainerStyle={{ paddingHorizontal: SPACING.lg, paddingBottom: 120 }}
                    renderItem={({ item }) => {
                        const isActive = currentTrack?.id === item.id;
                        return (
                            <TouchableOpacity
                                style={[styles.row, RETRO_SHADOW_SM, isActive && styles.rowActive]}
                                onPress={() => handlePlay(item)}
                                activeOpacity={0.8}
                            >
                                <Image source={{ uri: item.thumbnail }} style={styles.thumb} />
                                <View style={styles.info}>
                                    <Text style={[styles.title, isActive && { color: COLORS.secondary }]} numberOfLines={1}>
                                        {item.title}
                                    </Text>
                                    <Text style={styles.duration}>{formatDuration(item.duration)}</Text>
                                </View>
                                <TouchableOpacity
                                    style={[styles.playBtn, isActive && styles.playBtnActive]}
                                    onPress={() => handlePlay(item)}
                                >
                                    <MaterialIcons
                                        name={isActive ? 'pause' : 'play-arrow'}
                                        size={22}
                                        color={isActive ? COLORS.textLight : COLORS.textDark}
                                    />
                                </TouchableOpacity>
                                <TouchableOpacity style={styles.deleteBtn} onPress={() => handleDelete(item)}>
                                    <MaterialIcons name="delete-outline" size={20} color={COLORS.red} />
                                </TouchableOpacity>
                            </TouchableOpacity>
                        );
                    }}
                />
            )}
        </View>
    );
};

const styles = StyleSheet.create({
    container: { flex: 1, backgroundColor: COLORS.background },
    header: {
        flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center',
        paddingHorizontal: SPACING.lg, paddingTop: SPACING.xxl, paddingBottom: SPACING.sm,
    },
    backBtn: {
        width: 36, height: 36, borderRadius: 18, backgroundColor: COLORS.primary,
        ...RETRO_BORDER, justifyContent: 'center', alignItems: 'center',
    },
    headerTitle: { fontSize: FONT_SIZE.xl, fontWeight: '700', color: COLORS.textDark },
    trackCount: {
        fontSize: FONT_SIZE.sm, color: COLORS.textMuted, fontWeight: '500',
        paddingHorizontal: SPACING.lg, marginBottom: SPACING.md,
    },
    emptyContainer: {
        flex: 1, justifyContent: 'center', alignItems: 'center', paddingBottom: SPACING.xxl * 2,
    },
    emptyTitle: { fontSize: FONT_SIZE.lg, fontWeight: '700', color: COLORS.textDark, marginTop: SPACING.md },
    emptySubtitle: { fontSize: FONT_SIZE.md, color: COLORS.textMuted, textAlign: 'center', marginTop: SPACING.sm, lineHeight: 22 },
    row: {
        flexDirection: 'row', alignItems: 'center', backgroundColor: COLORS.card,
        ...RETRO_BORDER, borderRadius: RADIUS.xl, padding: SPACING.md, marginBottom: SPACING.md,
    },
    rowActive: { backgroundColor: COLORS.secondary + '15' },
    thumb: {
        width: 52, height: 52, borderRadius: RADIUS.md, ...RETRO_BORDER,
        backgroundColor: COLORS.secondary + '20',
    },
    info: { flex: 1, marginLeft: SPACING.md },
    title: { fontSize: FONT_SIZE.md, fontWeight: '700', color: COLORS.textDark },
    duration: { color: COLORS.textMuted, fontSize: FONT_SIZE.sm, marginTop: 2 },
    playBtn: {
        width: 40, height: 40, borderRadius: 20, backgroundColor: COLORS.card,
        ...RETRO_BORDER, justifyContent: 'center', alignItems: 'center',
    },
    playBtnActive: { backgroundColor: COLORS.secondary },
    deleteBtn: { padding: SPACING.sm, marginLeft: SPACING.xs },
});

export default PlaylistScreen;
