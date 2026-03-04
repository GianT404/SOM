import React, { useCallback, useState } from 'react';
import {
    View, Text, ScrollView, TextInput, TouchableOpacity, StyleSheet, StatusBar, Image
} from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';
import {
    COLORS,
    searchBar,
    NEO_BRUTALISM,
    searchInput,
    searchIcon,
    SPACING,
    RADIUS,
    FONT_SIZE,
    HEADER,
    headerTitleContainer,
    avatar,
    headerTitle,
    headerLeft,
    RETRO_SHADOW_SM
} from '../theme';
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
            {/* Header */}
            <View style={[HEADER, { backgroundColor: COLORS.background }]}>
                <View style={[headerLeft, { zIndex: 1 }]}>
                    <TouchableOpacity
                        style={[avatar]}
                        onPress={() => navigation.navigate('Settings')}
                        activeOpacity={0.8}
                    >
                        <Image source={require('../../assets/logo.png')} style={avatar} />
                    </TouchableOpacity>
                </View>
                <View style={headerTitleContainer}>
                    <Text style={headerTitle}>Home1</Text>
                </View>
            </View>
            <StatusBar barStyle="dark-content" backgroundColor={COLORS.background} />
            <ScrollView showsVerticalScrollIndicator={false}
            >
                {/* Hero Text */}
                <View style={styles.hero}>
                    {/* <Text style={styles.heroTitle}>Listening Everyday</Text> */}
                    <Text style={styles.heroSubtitle}>Explore millions of music according to your taste</Text>
                </View>
                <View style={styles.searchWrapper}>
                    <View style={[
                        NEO_BRUTALISM.shadowSm,
                        { backgroundColor: COLORS.primary, borderRadius: RADIUS.sm }
                    ]} />
                    <View style={[searchBar, { marginHorizontal: 0, marginBottom: 0 }]}>
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

                {/* Music Card Grid */}
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
    container: { flex: 1, backgroundColor: COLORS.background },
    searchWrapper: {
        marginHorizontal: SPACING.lg,
        marginBottom: SPACING.lg,
        position: 'relative',
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
    gridContainer: {
        flexDirection: 'row',
        flexWrap: 'wrap',
        justifyContent: 'space-between',
        paddingHorizontal: 20,
        marginTop: 20,
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