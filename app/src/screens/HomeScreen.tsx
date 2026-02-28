import React, { useCallback, useState } from 'react';
import {
    View, Text, ScrollView, TextInput, TouchableOpacity, StyleSheet, StatusBar,
} from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';
import { COLORS, searchBar, searchInput, searchIcon, SPACING, RADIUS, FONT_SIZE, RETRO_BORDER, RETRO_SHADOW, RETRO_SHADOW_SM, HEADER, headerTitleContainer, avatar, headerTitle, headerLeft } from '../theme';
import { usePlayer } from '../contexts/PlayerContext';
import MusicCard from '../components/MusicCard';
import MiniPlayer from '../components/MiniPlayer';

import { getPlaylist, OfflineTrack } from '../services/playlistStore';
import { useFocusEffect } from '@react-navigation/native';

const HomeScreen: React.FC<{ navigation: any }> = ({ navigation }) => {
    const { play, currentTrack } = usePlayer();
    const [tracks, setTracks] = useState<OfflineTrack[]>([]);

    useFocusEffect(
        useCallback(() => {
            getPlaylist().then(setTracks);
        }, [])
    );

    const handlePlay = useCallback((track: OfflineTrack) => {
        play(track);
        navigation.navigate('NowPlaying');
    }, [play, navigation]);

    return (
        <View style={styles.container}>
            <StatusBar barStyle="dark-content" backgroundColor={COLORS.background} />
            <ScrollView showsVerticalScrollIndicator={false}>
                {/* Header */}
                <View style={HEADER}>
                    <View style={headerLeft}>
                        <TouchableOpacity
                            style={[RETRO_SHADOW_SM, avatar]}
                            onPress={() => navigation.navigate('Settings')}
                        >
                            <MaterialIcons name="person" size={22} color={COLORS.primary} />
                        </TouchableOpacity>
                    </View>
                    <View style={headerTitleContainer}>
                        <Text style={headerTitle}>Home</Text>
                    </View>
                </View>

                {/* Hero Text */}
                <View style={styles.hero}>
                    <Text style={styles.heroTitle}>Listening Everyday</Text>
                    <Text style={styles.heroSubtitle}>Explore millions of music according to your taste</Text>
                </View>

                {/* Search Bar */}
                <View style={[searchBar, RETRO_SHADOW_SM]}>
                    <TextInput
                        style={searchInput}
                        placeholder="Muốn gì!?"
                        placeholderTextColor={COLORS.textMuted}
                        onFocus={() => navigation.navigate('Search')}
                    />
                    <TouchableOpacity style={searchIcon}>
                        <MaterialIcons name="search" size={22} color={COLORS.textDark} />
                    </TouchableOpacity>
                </View>

                {/* Tab Row */}
                <View style={styles.tabRow}>
                    <View style={styles.tabActive}>
                        <Text style={styles.tabActiveText}>Overview</Text>
                        <View style={styles.tabUnderline} />
                    </View>
                    <Text style={styles.tabText}>Songs</Text>
                    <Text style={styles.tabText}>Album</Text>
                    <Text style={styles.tabText}>Artist</Text>
                </View>

                {/* Music Card Grid — 2 columns */}
                {tracks.length === 0 ? (
                    <View style={styles.emptyContainer}>
                        <MaterialIcons name="music-note" size={48} color={COLORS.textMuted} />
                        <Text style={styles.emptyText}>Trống như đường tình bạn vậy?</Text>
                        <Text style={styles.emptySubtext}>Tìm kiếm, thêm nhạc, chill thôi nào.</Text>
                    </View>
                ) : (
                    <View style={styles.gridContainer}>
                        {tracks.map((track: OfflineTrack, i: number) => (
                            <MusicCard
                                key={track.id}
                                {...track}
                                index={i}
                                onPress={() => handlePlay(track)}
                            />
                        ))}
                    </View>
                )}


                <View style={{ height: 140 }} />
            </ScrollView>

            {currentTrack && (
                <View style={styles.miniPlayerContainer}>
                    <MiniPlayer onPress={() => navigation.navigate('NowPlaying')} />
                </View>
            )}
        </View>
    );
};

const styles = StyleSheet.create({
    gridContainer: {
        flexDirection: 'row',
        flexWrap: 'wrap',
        justifyContent: 'space-between',
        paddingHorizontal: 20,
        marginTop: 20,
    },
    container: { flex: 1, backgroundColor: COLORS.background },



    headerRight: {
        minWidth: 44,
        alignItems: 'flex-end',
    },
    menuBtn: {
        width: 44, height: 44, backgroundColor: COLORS.secondary, borderRadius: RADIUS.md,
        ...RETRO_BORDER, justifyContent: 'center', alignItems: 'center',
    },
    menuGrid: { flexDirection: 'row', flexWrap: 'wrap', width: 14, gap: 3 },
    menuDot: {
        width: 5, height: 5, borderRadius: 3, backgroundColor: COLORS.card,
        borderWidth: 1, borderColor: COLORS.border,
    },
    notifBtn: { position: 'relative' as const, padding: 2 },
    notifDot: {
        position: 'absolute' as const, top: 3, right: 3, width: 8, height: 8,
        backgroundColor: COLORS.red, borderRadius: 4, borderWidth: 2, borderColor: COLORS.background,
    },
    hero: { paddingHorizontal: SPACING.lg, marginTop: SPACING.md, marginBottom: SPACING.lg },
    heroTitle: { fontSize: 26, fontWeight: '900', color: COLORS.textDark },
    heroSubtitle: {
        fontSize: FONT_SIZE.sm, color: COLORS.textMuted, fontWeight: '400', marginTop: SPACING.xs,
    },

    tabRow: {
        flexDirection: 'row', paddingHorizontal: SPACING.lg, gap: SPACING.xl,
        marginBottom: SPACING.md, alignItems: 'center',
    },
    tabActive: { alignItems: 'center' },
    tabActiveText: {
        fontSize: FONT_SIZE.md, fontWeight: '600', color: COLORS.textDark, marginBottom: 4,
    },
    tabUnderline: {
        width: '100%' as any, height: 2, backgroundColor: COLORS.textDark, borderRadius: 1,
    },
    tabText: { fontSize: FONT_SIZE.md, fontWeight: '400', color: COLORS.textMuted },
    cardGrid: {
        flexDirection: 'row', flexWrap: 'wrap', paddingHorizontal: SPACING.sm + 2,
    },
    emptyContainer: {
        alignItems: 'center', justifyContent: 'center', marginTop: 60, paddingHorizontal: SPACING.xl,
    },
    emptyText: {
        fontSize: FONT_SIZE.lg, fontWeight: '700', color: COLORS.textDark, marginTop: SPACING.md,
    },
    emptySubtext: {
        fontSize: FONT_SIZE.sm, color: COLORS.textMuted, textAlign: 'center', marginTop: SPACING.xs, lineHeight: 20,
    },
    miniPlayerContainer: {
        position: 'absolute',
        bottom: 24,
        left: 0,
        right: 0,
    },
});

export default HomeScreen;
