import React, { useCallback, useState } from 'react';
import {
    View, Text, ScrollView, TextInput, TouchableOpacity, StyleSheet, StatusBar,
} from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';
import { COLORS, searchBar, searchInput, searchIcon, SPACING, RADIUS, FONT_SIZE, RETRO_BORDER, RETRO_SHADOW, RETRO_SHADOW_SM, HEADER, headerTitleContainer, avatar, headerTitle, headerLeft } from '../theme';
import { usePlayer } from '../contexts/PlayerContext';
import MusicCard from '../components/MusicCard';
import MiniPlayer from '../components/MiniPlayer';

import { getPlaylist, getDeletedPlaylist, softDeleteTrack, restoreTrack, permanentlyDeleteTrack, OfflineTrack } from '../services/playlistStore';
import { useFocusEffect } from '@react-navigation/native';

const HomeScreen: React.FC<{ navigation: any }> = ({ navigation }) => {
    const { play, currentTrack } = usePlayer();
    const [tracks, setTracks] = useState<OfflineTrack[]>([]);
    const [deletedTracks, setDeletedTracks] = useState<OfflineTrack[]>([]);
    const [activeTab, setActiveTab] = useState<'playlist' | 'deleted'>('playlist');
    const [isDeleteMode, setIsDeleteMode] = useState<boolean>(false);

    const loadTracks = useCallback(() => {
        getPlaylist().then(setTracks);
        getDeletedPlaylist().then(setDeletedTracks);
    }, []);

    useFocusEffect(
        useCallback(() => {
            loadTracks();
            setIsDeleteMode(false);
        }, [loadTracks])
    );

    const handlePlay = useCallback((track: OfflineTrack) => {
        play(track);
        navigation.navigate('NowPlaying');
    }, [play, navigation]);

    const handleSoftDelete = async (id: string) => {
        await softDeleteTrack(id);
        loadTracks();
        setIsDeleteMode(false);
    };

    const handleRestore = async (id: string) => {
        await restoreTrack(id);
        loadTracks();
        setIsDeleteMode(false);
    };

    const handlePermanentDelete = async (id: string) => {
        await permanentlyDeleteTrack(id);
        loadTracks();
        setIsDeleteMode(false);
    };

    const renderTracks = activeTab === 'playlist' ? tracks : deletedTracks;

    return (
        <View style={styles.container} onStartShouldSetResponder={() => {
            if (isDeleteMode) setIsDeleteMode(false);
            return false;
        }}>
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
                    <Text style={styles.heroTitle}>Listening Eveeryday</Text>
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
                    <TouchableOpacity
                        style={activeTab === 'playlist' ? styles.tabActive : undefined}
                        onPress={() => { setActiveTab('playlist'); setIsDeleteMode(false); }}
                        activeOpacity={0.8}
                    >
                        <Text style={activeTab === 'playlist' ? styles.tabActiveText : styles.tabText}>Playlist</Text>
                        {activeTab === 'playlist' && <View style={styles.tabUnderline} />}
                    </TouchableOpacity>
                    <TouchableOpacity
                        style={activeTab === 'deleted' ? styles.tabActive : undefined}
                        onPress={() => { setActiveTab('deleted'); setIsDeleteMode(false); }}
                        activeOpacity={0.8}
                    >
                        <Text style={activeTab === 'deleted' ? styles.tabActiveText : styles.tabText}>Đã xóa</Text>
                        {activeTab === 'deleted' && <View style={styles.tabUnderline} />}
                    </TouchableOpacity>
                </View>

                {/* Music Card Grid — 2 columns */}
                {renderTracks.length === 0 ? (
                    <View style={styles.emptyContainer}>
                        <MaterialIcons name={activeTab === 'playlist' ? "music-note" : "delete-outline"} size={48} color={COLORS.textMuted} />
                        <Text style={styles.emptyText}>{activeTab === 'playlist' ? 'Trống như đường tình bạn vậy?' : 'Thùng rác trống'}</Text>
                        <Text style={styles.emptySubtext}>{activeTab === 'playlist' ? 'Tìm kiếm, thêm nhạc, chill thôi nào.' : 'Không có bài hát nào bị xóa'}</Text>
                    </View>
                ) : (
                    <View style={styles.gridContainer}>
                        {renderTracks.map((track: OfflineTrack, i: number) => (
                            <MusicCard
                                key={track.id}
                                {...track}
                                index={i}
                                showDeleteBtn={isDeleteMode}
                                onPress={() => {
                                    if (isDeleteMode) {
                                        setIsDeleteMode(false);
                                    } else {
                                        handlePlay(track);
                                    }
                                }}
                                onLongPress={() => {
                                    setIsDeleteMode(true);
                                }}
                                onDeletePress={() => {
                                    if (activeTab === 'playlist') {
                                        handleSoftDelete(track.id);
                                    } else {
                                        handleRestore(track.id);
                                    }
                                }}
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
