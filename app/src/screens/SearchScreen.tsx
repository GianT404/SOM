import React, { useState, useCallback } from 'react';
import {
    View, Text, TextInput, FlatList, TouchableOpacity, StyleSheet, ActivityIndicator, StatusBar,
} from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';
import {
    COLORS,
    SPACING,
    RADIUS,
    FONT_SIZE,
    NEO_BRUTALISM,
    HEADER,
    headerTitle,
    backBtn,
    headerTitleContainer,
    headerLeft,
    searchBar,
    searchInput,
    searchIcon
} from '../theme';
import { usePlayer } from '../contexts/PlayerContext';
import api, { SearchResult } from '../services/api';
import SongRow from '../components/SongRow';
import { NeoShadowWrapper } from '../components/NeoShadowWrapper';

const SearchScreen: React.FC<{ navigation: any }> = ({ navigation }) => {
    const { play } = usePlayer();
    const [query, setQuery] = useState('');
    const [results, setResults] = useState<SearchResult[]>([]);
    const [loading, setLoading] = useState(false);
    const [searched, setSearched] = useState(false);

    const handleSearch = useCallback(async (q?: string) => {
        const term = (q || query).trim();
        if (!term) return;
        setQuery(term);
        setLoading(true);
        setSearched(true);
        try {
            const data = await api.search(term);
            setResults(data || []);
        } catch { setResults([]); }
        finally { setLoading(false); }
    }, [query]);

    const handlePlay = useCallback((item: SearchResult) => {
        play({ id: item.id, title: item.title, thumbnail: item.thumbnail, duration: item.duration, uploader: item.uploader });
        navigation.navigate('NowPlaying');
    }, [play, navigation]);

    return (
        <View style={styles.container}>
            <StatusBar barStyle="dark-content" backgroundColor={COLORS.background} />
            <View style={HEADER}>
                <View style={headerLeft}>
                    <TouchableOpacity onPress={() => navigation.goBack()} activeOpacity={0.8}>
                        <NeoShadowWrapper borderRadius={RADIUS.sm} offset={3} style={backBtn}>
                            <MaterialIcons name="arrow-back" size={24} color={COLORS.textDark} />
                        </NeoShadowWrapper>
                    </TouchableOpacity>
                </View>
                <View style={headerTitleContainer} pointerEvents="none">
                    <Text style={headerTitle}>Tìm kiếm</Text>
                </View>
            </View>

            <View style={styles.searchWrapper}>
                <View style={[
                    NEO_BRUTALISM.shadowSm,
                    { backgroundColor: COLORS.primary, borderRadius: RADIUS.sm }
                ]} />
                <View style={[searchBar, { marginHorizontal: 0, marginBottom: 0 }]}>
                    <TextInput
                        style={searchInput}
                        placeholder="Artists, songs, or podcasts"
                        placeholderTextColor={COLORS.textMuted}
                        value={query}
                        onChangeText={(text) => {
                            setQuery(text);
                            if (text === '') {
                                setResults([]);
                                setSearched(false);
                            }
                        }}
                        onSubmitEditing={() => handleSearch()}
                        returnKeyType="search"
                    />

                    <TouchableOpacity
                        style={searchIcon}
                        onPress={() => {
                            if (query.length > 0) {
                                setQuery('');
                                setResults([]);
                                setSearched(false);
                            } else {
                                handleSearch();
                            }
                        }}
                    >
                        <MaterialIcons
                            name={query.length > 0 ? "close" : "search"}
                            size={22}
                            color={COLORS.textDark}
                        />
                    </TouchableOpacity>
                </View>
            </View>

            {!searched ? (
                <ScrollContent>
                    <View style={styles.sectionHeader}>
                    </View>
                </ScrollContent>
            ) : loading ? (
                <View style={styles.loadingCenter}>
                    <ActivityIndicator size="large" color={COLORS.primary} />
                </View>
            ) : (
                <FlatList
                    data={results}
                    keyExtractor={(item) => item.id}
                    contentContainerStyle={{ paddingHorizontal: SPACING.lg, paddingBottom: 120 }}
                    ListEmptyComponent={
                        <View style={styles.loadingCenter}>
                            <MaterialIcons name="music-off" size={48} color={COLORS.textMuted} />
                            <Text style={styles.emptyText}>No results found</Text>
                        </View>
                    }
                    renderItem={({ item }) => (
                        <SongRow
                            title={item.title}
                            subtitle={item.uploader}
                            thumbnail={item.thumbnail}
                            duration={item.duration}
                            onPress={() => handlePlay(item)}
                        />
                    )}
                />
            )}
        </View>
    );
};

const ScrollContent: React.FC<{ children: React.ReactNode }> = ({ children }) => (
    <FlatList data={[]} renderItem={() => null} ListHeaderComponent={<>{children}</>} contentContainerStyle={{ paddingBottom: 120 }} />
);

const styles = StyleSheet.create({
    container: { flex: 1, backgroundColor: COLORS.background },
    searchWrapper: {
        marginHorizontal: SPACING.lg,
        marginBottom: SPACING.lg,
        marginTop: SPACING.md,
        position: 'relative',
    },
    sectionHeader: {
        flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center',
        paddingHorizontal: SPACING.lg, marginTop: SPACING.lg, marginBottom: SPACING.md,
    },
    sectionTitle: { fontSize: FONT_SIZE.lg, fontWeight: '700', color: COLORS.textDark },
    genreGrid: { flexDirection: 'row', flexWrap: 'wrap', paddingHorizontal: SPACING.lg, gap: SPACING.md },
    genreCard: {
        width: '46%' as any, aspectRatio: 1.2, borderWidth: 2, borderColor: COLORS.border, borderRadius: RADIUS.xl,
        justifyContent: 'flex-end', padding: SPACING.md,
    },
    genreLabel: { fontSize: FONT_SIZE.xl, fontWeight: '900' },
    loadingCenter: { flex: 1, justifyContent: 'center', alignItems: 'center', paddingTop: SPACING.xxl * 2 },
    emptyText: { color: COLORS.textMuted, marginTop: SPACING.md, fontSize: FONT_SIZE.md },
});

export default SearchScreen;