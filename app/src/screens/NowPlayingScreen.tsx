import React, { useCallback, useEffect, useState, useRef } from 'react';
import {
    View, Text, Image, TouchableOpacity, StyleSheet, StatusBar, Alert, Modal,
    TouchableWithoutFeedback, ScrollView, Dimensions,
    StyleProp, ViewStyle, GestureResponderEvent
} from 'react-native';
import { MaterialIcons, Ionicons, Feather } from '@expo/vector-icons';
import { usePlayer } from '../contexts/PlayerContext';
import { downloadAndAdd, DownloadProgress, isDownloaded } from '../services/playlistStore';
import SyncedLyrics from '../components/SyncedLyrics';
import { NeoShadowWrapper } from '../components/NeoShadowWrapper';
import { COLORS, RADIUS } from '../theme';
import MarqueeText from '../components/MarqueeText';

const { width: SCREEN_WIDTH } = Dimensions.get('window');

interface NeoShadowWrapperProps {
    children: React.ReactNode;
    style?: StyleProp<ViewStyle>;
    containerStyle?: StyleProp<ViewStyle>;
    borderRadius?: number;
    offset?: number;
}

const NowPlayingScreen = ({ navigation }: { navigation: any }) => {
    const { currentTrack, isPlaying, pause, resume, isLoading, position, duration, seekTo, skipToNext, skipToPrevious, isShuffle, toggleShuffle, repeatMode, toggleRepeat } = usePlayer();

    const [showMenu, setShowMenu] = useState<boolean>(false);
    const [dlState, setDlState] = useState<'idle' | 'downloading' | 'done'>('idle');
    const [dlProgressText, setDlProgressText] = useState<string>('');
    const [selectedLanguage, setSelectedLanguage] = useState<string>('vi'); // Default language
    const [availableLanguages, setAvailableLanguages] = useState<string[]>([]);

    // --- LOGIC TUA NHẠC (SEEKING) ---
    const [waveWidth, setWaveWidth] = useState<number>(0);
    const [isSeeking, setIsSeeking] = useState<boolean>(false);
    const [seekProgress, setSeekProgress] = useState<number>(0);

    const displayProgress = isSeeking ? seekProgress : (duration > 0 ? position / duration : 0);
    const displayPosition = isSeeking ? seekProgress * duration : position;

    // --- LOGIC ANIMATION WAVEFORM GIẢ LẬP HIỆU NĂNG CAO ---
    // (Removed: replaced with simple progress bar)


    const formatTime = (ms: number) => {
        if (!ms || isNaN(ms)) return "00:00";
        const s = Math.floor(ms / 1000);
        return `${Math.floor(s / 60).toString().padStart(2, '0')}:${(s % 60).toString().padStart(2, '0')}`;
    };

    const handleSeekTouch = (evt: GestureResponderEvent) => {
        if (waveWidth === 0 || duration === 0) return;
        const x = evt.nativeEvent.locationX;
        let p = x / waveWidth;
        if (p < 0) p = 0;
        if (p > 1) p = 1;
        setSeekProgress(p);
        return p;
    };
    useEffect(() => {
        if (!currentTrack) return;
        isDownloaded(currentTrack.id).then(yes => {
            setDlState(yes ? 'done' : 'idle');
        });
    }, [currentTrack?.id]);

    const handleDownload = useCallback(async () => {
        if (!currentTrack || dlState !== 'idle') return;
        setDlState('downloading');
        setDlProgressText('Đang chuẩn bị...');
        setShowMenu(false);

        try {
            await downloadAndAdd(currentTrack, (p: DownloadProgress) => {
                if (p.totalBytesExpectedToWrite === -1) {
                    // Server is still running yt-dlp — no bytes yet
                    setDlProgressText('Đang chuẩn bị...');
                } else if (p.totalBytesExpectedToWrite > 0) {
                    const pct = Math.round((p.totalBytesWritten / p.totalBytesExpectedToWrite) * 100);
                    setDlProgressText(`${pct}%`);
                } else {
                    // Content-Length unknown — show bytes transferred
                    const mb = (p.totalBytesWritten / 1024 / 1024).toFixed(1);
                    setDlProgressText(`${mb} MB`);
                }
            });
            setDlState('done');
        } catch (err: any) {
            Alert.alert('Lỗi tải xuống', err?.message || 'Kiểm tra lại kết nối mạng');
            setDlState('idle');
        }
    }, [currentTrack, dlState]);

    if (!currentTrack) {
        return (
            <View style={[styles.container, styles.emptyCenter]}>
                <MaterialIcons name="music-off" size={64} color="#A0A0A0" />
                <Text style={styles.emptyText}>Chưa có bài hát nào phát</Text>
                <TouchableOpacity onPress={() => navigation.goBack()}>
                    <Text style={styles.goBackText}>Quay lại</Text>
                </TouchableOpacity>
            </View>
        );
    }

    return (
        <View style={styles.container}>
            <StatusBar barStyle="dark-content" backgroundColor="#F8F9F5" />

            <View style={styles.header}>
                <TouchableOpacity onPress={() => navigation.goBack()} activeOpacity={0.8}>
                    <NeoShadowWrapper borderRadius={RADIUS.sm} offset={3} style={[styles.headerBtn, styles.btnOrange]}>
                        <MaterialIcons name="keyboard-arrow-down" size={26} color="#1A1A1A" />
                    </NeoShadowWrapper>
                </TouchableOpacity>

                <View style={styles.headerCenter}>

                    <MarqueeText
                        text={currentTrack.title}
                        style={styles.trackTitle}

                    />
                    <Text style={styles.trackArtist}>{currentTrack.uploader || "Unknown Artist"}</Text>
                </View>

                <TouchableOpacity onPress={() => setShowMenu(true)} activeOpacity={0.8}>
                    <NeoShadowWrapper borderRadius={RADIUS.sm} offset={3} style={[styles.headerBtn, styles.btnWhite]}>
                        <MaterialIcons name="more-horiz" size={26} color="#1A1A1A" />
                    </NeoShadowWrapper>
                </TouchableOpacity>
            </View>

            <Modal visible={showMenu} transparent animationType="fade">
                <TouchableWithoutFeedback onPress={() => setShowMenu(false)}>
                    <View style={styles.modalOverlay}>
                        <NeoShadowWrapper borderRadius={RADIUS.sm} offset={4} style={styles.dropdownMenu} containerStyle={styles.dropdownMenuContainer}>
                            <TouchableOpacity style={styles.menuItem} onPress={handleDownload} disabled={dlState !== 'idle'}>
                                <MaterialIcons
                                    name={dlState === 'done' ? 'check-circle' : 'file-download'}
                                    size={24}
                                    color={dlState === 'done' ? '#10B981' : '#1A1A1A'}
                                />
                                <Text style={[styles.menuItemText, dlState === 'done' && { color: '#10B981' }]}>
                                    {dlState === 'done' ? 'Đã tải' : dlState === 'downloading' ? `Đang tải ${dlProgressText}` : 'Lưu bài hát'}
                                </Text>
                            </TouchableOpacity>
                            {/* <TouchableOpacity style={styles.menuItem}>
                                <MaterialIcons name="share" size={24} color="#1A1A1A" />
                                <Text style={styles.menuItemText}>Chia sẻ</Text>
                            </TouchableOpacity> */}
                            {availableLanguages.length > 0 && <View style={{ height: 1, backgroundColor: '#E0E0E0', marginVertical: 4 }} />}
                            {availableLanguages.map(lang => (
                                <TouchableOpacity
                                    key={lang}
                                    style={styles.menuItem}
                                    onPress={() => {
                                        setSelectedLanguage(lang);
                                        setShowMenu(false);
                                    }}
                                >
                                    <MaterialIcons
                                        name={selectedLanguage === lang ? "radio-button-checked" : "radio-button-unchecked"}
                                        size={24}
                                        color="#1A1A1A"
                                    />
                                    <Text style={styles.menuItemText}>Lyric: {lang.toUpperCase()}</Text>
                                </TouchableOpacity>
                            ))}
                        </NeoShadowWrapper>
                    </View>
                </TouchableWithoutFeedback>
            </Modal>
            {/* thumbnail */}
            <View style={styles.artContainer}>
                <ScrollView horizontal pagingEnabled showsHorizontalScrollIndicator={false} style={{ flex: 1, width: SCREEN_WIDTH }}>
                    <View style={{ width: SCREEN_WIDTH, alignItems: 'center', justifyContent: 'center' }}>
                        <NeoShadowWrapper borderRadius={RADIUS.sm} offset={6} style={styles.artWrapper}>
                            <Image source={{ uri: currentTrack.thumbnail }} style={styles.albumArt} />
                        </NeoShadowWrapper>
                    </View>
                    <View style={{ width: SCREEN_WIDTH, height: 280 }}>
                        <SyncedLyrics
                            track={currentTrack}
                            position={displayPosition}
                            selectedLanguage={selectedLanguage}
                            onLanguagesLoaded={setAvailableLanguages}
                            isOfflineTrack={!!(currentTrack.localUri && currentTrack.lyrics)}
                        />
                    </View>
                </ScrollView>
            </View>

            <NeoShadowWrapper borderRadius={RADIUS.sm} offset={6} containerStyle={{ marginHorizontal: 24, marginBottom: 30 }} style={styles.controlBox}>
                <View style={styles.waveformRow}>
                    <Text style={styles.timeText}>{formatTime(displayPosition)}</Text>

                    <View
                        style={styles.waveformTouchArea}
                        onLayout={(e) => setWaveWidth(e.nativeEvent.layout.width)}
                        onStartShouldSetResponder={() => true}
                        onResponderGrant={(e) => {
                            setIsSeeking(true);
                            handleSeekTouch(e);
                        }}
                        onResponderMove={handleSeekTouch}
                        onResponderRelease={(e) => {
                            const finalProgress = handleSeekTouch(e);
                            setIsSeeking(false);
                            if (finalProgress !== undefined && seekTo) seekTo(finalProgress * duration);
                        }}
                    >
                        <View style={styles.progressTrack} pointerEvents="none">
                            <View style={[styles.progressFill, { width: `${displayProgress * 100}%` }]} />
                            <View style={styles.progressKnob} />
                        </View>
                    </View>

                    <Text style={styles.timeText}>{formatTime(duration)}</Text>
                </View>

                <View style={styles.mainControls}>
                    <TouchableOpacity onPress={toggleShuffle} activeOpacity={0.7}>
                        <Ionicons name="shuffle" size={24} color={isShuffle ? COLORS.primary : "#A0A0A0"} />
                    </TouchableOpacity>

                    <TouchableOpacity onPress={skipToPrevious} activeOpacity={0.7}>
                        <MaterialIcons name="skip-previous" size={32} color="#1A1A1A" />
                    </TouchableOpacity>

                    <TouchableOpacity onPress={isPlaying ? pause : resume} activeOpacity={0.8}>
                        <NeoShadowWrapper borderRadius={RADIUS.sm} offset={4} style={styles.playPauseBtn}>
                            {isLoading ? (
                                <MaterialIcons name="hourglass-empty" size={30} color="#1A1A1A" />
                            ) : (
                                <Ionicons name={isPlaying ? "pause" : "play"} size={28} color="#1A1A1A" style={{ marginLeft: isPlaying ? 0 : 4 }} />
                            )}
                        </NeoShadowWrapper>
                    </TouchableOpacity>
                    <TouchableOpacity onPress={skipToNext} activeOpacity={0.7}>
                        <MaterialIcons name="skip-next" size={32} color="#1A1A1A" />
                    </TouchableOpacity>

                    <TouchableOpacity onPress={toggleRepeat} activeOpacity={0.7}>
                        {repeatMode === 'one' ? (
                            <MaterialIcons name="repeat-one" size={22} color="#9D72FF" />
                        ) : (
                            <Feather name="repeat" size={20} color={repeatMode === 'all' ? "#9D72FF" : "#A0A0A0"} />
                        )}
                    </TouchableOpacity>
                </View>
            </NeoShadowWrapper>

            {dlState === 'downloading' && (
                <NeoShadowWrapper borderRadius={RADIUS.sm} offset={4} containerStyle={styles.miniProgressContainer}>
                    <View style={styles.miniProgress}>
                        <Text style={styles.miniProgressText}>Đang tải: {dlProgressText}</Text>
                    </View>
                </NeoShadowWrapper>
            )}
        </View>
    );
};

const styles = StyleSheet.create({
    container: { flex: 1, backgroundColor: '#F8F9F5' },

    header: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginTop: 60, marginBottom: 20, paddingHorizontal: 24 },
    headerBtn: { width: 44, height: 44, justifyContent: 'center', alignItems: 'center' },
    btnOrange: { backgroundColor: '#FFB86C' },
    btnWhite: { backgroundColor: '#FFF' },
    headerCenter: {
        flex: 1, // BẮT BUỘC: Để container chiếm hết khoảng trống ở giữa
        alignItems: 'center',
        justifyContent: 'center',
        marginHorizontal: 10,
        overflow: 'hidden', // Đảm bảo chữ chạy ra ngoài là biến mất luôn
    },
    playingFrom: { fontSize: 10, fontWeight: '700', color: '#A0A0A0', letterSpacing: 1, marginBottom: 2 },
    playlistName: { fontSize: 14, fontWeight: '800', color: '#1A1A1A' },

    artContainer: { alignItems: 'center', marginTop: 20, height: 340, width: SCREEN_WIDTH },
    artWrapper: { width: SCREEN_WIDTH * 0.78, height: SCREEN_WIDTH * 0.78, backgroundColor: '#FFF', padding: 12 },
    albumArt: { width: '100%', height: '100%', borderRadius: RADIUS.sm },

    trackDetailRow: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'flex-end', marginTop: 10, marginBottom: 30, paddingHorizontal: 24 },
    trackTextContainer: { flex: 1 },
    trackTitle: { fontSize: 18, fontWeight: '900', color: '#1A1A1A', marginBottom: 4, },
    trackArtist: { fontSize: 14, color: '#666', fontWeight: '600' },
    paginationDots: { flexDirection: 'row', alignItems: 'center', gap: 6, paddingBottom: 4 },
    dot: { width: 6, height: 6, borderRadius: RADIUS.sm },

    controlBox: { backgroundColor: '#FFF', paddingVertical: 24, paddingHorizontal: 20 },
    waveformRow: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', marginBottom: 24 },
    timeText: { fontSize: 11, fontWeight: '700', color: '#1A1A1A', width: 40, textAlign: 'center' },

    waveformTouchArea: { flex: 1, height: 40, justifyContent: 'center', paddingHorizontal: 4 },
    progressTrack: {
        height: 6,
        backgroundColor: '#E0E0E0',
        borderRadius: 3,
        flexDirection: 'row',
        alignItems: 'center',
        overflow: 'visible',
    },
    progressFill: {
        height: '100%',
        backgroundColor: '#FFB86C',
        borderRadius: 3,
    },
    progressKnob: {
        width: 16,
        height: 16,
        borderRadius: 8,
        backgroundColor: '#1A1A1A',
        marginLeft: -8,
        borderWidth: 2,
        borderColor: '#FFF',
    },

    mainControls: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', paddingHorizontal: 10 },
    playPauseBtn: { width: 64, height: 64, backgroundColor: '#FFB86C', justifyContent: 'center', alignItems: 'center' },

    modalOverlay: { flex: 1, backgroundColor: 'rgba(0,0,0,0.1)' },
    dropdownMenuContainer: { position: 'absolute', top: 110, right: 24 },
    dropdownMenu: { backgroundColor: '#FFF', padding: 8, minWidth: 160 },
    menuItem: { flexDirection: 'row', alignItems: 'center', paddingVertical: 12, paddingHorizontal: 12, gap: 12 },
    menuItemText: { fontSize: 15, fontWeight: '700', color: '#1A1A1A' },

    miniProgressContainer: { position: 'absolute', bottom: 40, alignSelf: 'center' },
    miniProgress: { backgroundColor: '#FFF', paddingHorizontal: 20, paddingVertical: 12 },
    miniProgressText: { fontWeight: '800', fontSize: 14, color: '#1A1A1A' },

    emptyCenter: { justifyContent: 'center', alignItems: 'center', flex: 1 },
    emptyText: { color: '#666', fontSize: 16, marginTop: 16, fontWeight: '600' },
    goBackText: { color: '#FFB86C', fontSize: 16, marginTop: 24, fontWeight: '700' },
});

export default NowPlayingScreen;