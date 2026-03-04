import React, { useState, useEffect, useCallback } from 'react';
import {
    View, Text, ScrollView, TouchableOpacity, Switch, StyleSheet, StatusBar, Image,
    Alert, Modal, TouchableWithoutFeedback,
} from 'react-native';
import { MaterialIcons, Ionicons } from '@expo/vector-icons';
import {
    COLORS, SPACING, RADIUS, FONT_SIZE, RETRO_SHADOW_SM,
    HEADER, headerTitleContainer, headerTitle, headerLeft, backBtn,
} from '../theme';
import { NeoShadowWrapper } from '../components/NeoShadowWrapper';
import {
    AudioSettings, BufferSize, SampleRate,
    BUFFER_SIZE_OPTIONS, SAMPLE_RATE_OPTIONS,
    getAudioSettings, saveAudioSettings,
} from '../services/audioSettings';
import Constants from 'expo-constants';

// ─── Reusable Setting Row ──────────────────────────────────────────
interface SettingRowProps {
    icon: string;
    label: string;
    subtitle?: string;
    value?: boolean;
    onToggle?: (val: boolean) => void;
    onPress?: () => void;
    iconBg?: string;
    iconColor?: string;
    iconSet?: 'material' | 'ionicons';
}

const SettingRow: React.FC<SettingRowProps> = ({
    icon, label, subtitle, value, onToggle, onPress, iconBg, iconColor, iconSet = 'material',
}) => (
    <TouchableOpacity style={styles.settingRow} onPress={onPress} activeOpacity={onToggle ? 1 : 0.7}>
        <View style={[styles.settingIcon, { backgroundColor: iconBg || COLORS.primary + '25' }]}>
            {iconSet === 'ionicons' ? (
                <Ionicons name={icon as any} size={20} color={iconColor || COLORS.textDark} />
            ) : (
                <MaterialIcons name={icon as any} size={20} color={iconColor || COLORS.textDark} />
            )}
        </View>
        <View style={styles.settingInfo}>
            <Text style={styles.settingLabel}>{label}</Text>
            {subtitle && <Text style={styles.settingSubtitle}>{subtitle}</Text>}
        </View>
        {onToggle !== undefined && value !== undefined ? (
            <Switch
                value={value}
                onValueChange={onToggle}
                trackColor={{ false: '#D1D5DB', true: COLORS.primary }}
                thumbColor={value ? '#FFF' : '#F3F4F6'}
            />
        ) : (
            <MaterialIcons name="chevron-right" size={22} color={COLORS.textMuted} />
        )}
    </TouchableOpacity>
);

// ─── Neo-Brutalism Segmented Control ───────────────────────────────
interface SegmentedControlProps<T extends string> {
    options: { key: T; label: string }[];
    selected: T;
    onSelect: (key: T) => void;
    accent?: string;
}

function SegmentedControl<T extends string>({ options, selected, onSelect, accent = COLORS.primary }: SegmentedControlProps<T>) {
    return (
        <View style={segStyles.container}>
            {options.map((opt, i) => {
                const isActive = opt.key === selected;
                return (
                    <TouchableOpacity
                        key={opt.key}
                        style={[
                            segStyles.segment,
                            isActive && { backgroundColor: accent },
                            i === 0 && segStyles.firstSegment,
                            i === options.length - 1 && segStyles.lastSegment,
                        ]}
                        onPress={() => onSelect(opt.key)}
                        activeOpacity={0.8}
                    >
                        <Text style={[
                            segStyles.segmentText,
                            isActive && segStyles.segmentTextActive,
                        ]}>
                            {opt.label}
                        </Text>
                    </TouchableOpacity>
                );
            })}
        </View>
    );
}

const segStyles = StyleSheet.create({
    container: {
        flexDirection: 'row',
        borderWidth: 2,
        borderColor: COLORS.border,
        borderRadius: RADIUS.sm,
        overflow: 'hidden',
    },
    segment: {
        flex: 1,
        paddingVertical: 10,
        alignItems: 'center',
        justifyContent: 'center',
        backgroundColor: COLORS.card,
        borderRightWidth: 1,
        borderColor: COLORS.border,
    },
    firstSegment: { borderTopLeftRadius: RADIUS.sm - 2, borderBottomLeftRadius: RADIUS.sm - 2 },
    lastSegment: { borderTopRightRadius: RADIUS.sm - 2, borderBottomRightRadius: RADIUS.sm - 2, borderRightWidth: 0 },
    segmentText: { fontSize: FONT_SIZE.xs, fontWeight: '700', color: COLORS.textMuted },
    segmentTextActive: { color: '#1A1A1A' },
});

// ─── Tooltip Info Button ───────────────────────────────────────────
const InfoButton: React.FC<{ onPress: () => void }> = ({ onPress }) => (
    <TouchableOpacity onPress={onPress} activeOpacity={0.7} style={styles.infoBtn}>
        <MaterialIcons name="info-outline" size={18} color={COLORS.textMuted} />
    </TouchableOpacity>
);

// ─── Main Screen ───────────────────────────────────────────────────
const SettingsScreen: React.FC<{ navigation: any }> = ({ navigation }) => {
    const [audioSettings, setAudioSettings] = useState<AudioSettings>({
        bufferSize: 'balanced',
        sampleRate: 'auto',
    });
    const [tooltipVisible, setTooltipVisible] = useState(false);
    const [tooltipContent, setTooltipContent] = useState({ title: '', body: '' });

    // Load saved settings
    useEffect(() => {
        getAudioSettings().then(setAudioSettings);
    }, []);

    const updateSetting = useCallback(async <K extends keyof AudioSettings>(
        key: K, value: AudioSettings[K]
    ) => {
        const updated = { ...audioSettings, [key]: value };
        setAudioSettings(updated);
        await saveAudioSettings(updated);
    }, [audioSettings]);

    const showTooltip = (title: string, body: string) => {
        setTooltipContent({ title, body });
        setTooltipVisible(true);
    };

    return (
        <View style={styles.container}>
            <StatusBar barStyle="dark-content" backgroundColor={COLORS.background} />

            {/* Header */}
            <View style={HEADER}>
                <View style={headerLeft}>
                    <TouchableOpacity onPress={() => navigation.goBack()} activeOpacity={0.8}>
                        <NeoShadowWrapper borderRadius={RADIUS.sm} offset={3} style={backBtn}>
                            <MaterialIcons name="arrow-back" size={22} color={COLORS.textDark} />
                        </NeoShadowWrapper>
                    </TouchableOpacity>
                </View>
                <View style={headerTitleContainer} pointerEvents="none">
                    <Text style={headerTitle}>Settings</Text>
                </View>
            </View>

            <ScrollView showsVerticalScrollIndicator={false} contentContainerStyle={{ paddingBottom: 100 }}>
                {/* Profile Card */}
                <View style={styles.profileSection}>
                    <NeoShadowWrapper borderRadius={RADIUS.sm} offset={4} style={styles.profileCard}>
                        <Image source={require('../../assets/logo.png')} style={styles.profileAvatar} />
                        <View style={styles.profileInfo}>
                            <Text style={styles.profileName}>GianT</Text>
                            <View style={styles.badgeRow}>
                                <View style={styles.badge}>
                                    <MaterialIcons name="star" size={12} color={COLORS.primary} />
                                    <Text style={styles.badgeText}>Premium</Text>
                                </View>
                            </View>
                        </View>
                        <MaterialIcons name="chevron-right" size={22} color={COLORS.textMuted} />
                    </NeoShadowWrapper>
                </View>

                {/* ═══ AUDIO ENGINE ═══ */}
                <View style={styles.sectionHeader}>
                    <Text style={styles.sectionTitle}>AUDIO ENGINE</Text>
                    <View style={styles.sectionBadge}>
                        <MaterialIcons name="tune" size={12} color={COLORS.secondary} />
                        <Text style={styles.sectionBadgeText}>Pro</Text>
                    </View>
                </View>

                <View style={styles.section}>
                    {/* Buffer Size */}
                    <View style={[styles.engineCard, RETRO_SHADOW_SM]}>
                        <View style={styles.engineCardHeader}>
                            <View style={[styles.settingIcon, { backgroundColor: COLORS.emerald + '20' }]}>
                                <MaterialIcons name="memory" size={20} color={COLORS.emerald} />
                            </View>
                            <View style={{ flex: 1, marginLeft: SPACING.md }}>
                                <Text style={styles.engineLabel}>Buffer Sizee</Text>
                                <Text style={styles.engineDesc}>
                                    {BUFFER_SIZE_OPTIONS.find(o => o.key === audioSettings.bufferSize)?.samples} samples
                                </Text>
                            </View>
                            <InfoButton onPress={() => showTooltip(
                                'Buffer Size',
                                '• Stable (2048): Ít bị gián đoạn, phù hợp Bluetooth/thiết bị cũ. Độ trễ cao hơn.\n\n• Balanced (1024): Cân bằng giữa ổn định và độ trễ. Khuyến nghị cho đa số thiết bị.\n\n• Fast (512): Độ trễ thấp nhất, phù hợp tai nghe có dây. Có thể bị ngắt trên thiết bị yếu.'
                            )} />
                        </View>
                        <View style={{ marginTop: SPACING.md }}>
                            <SegmentedControl
                                options={BUFFER_SIZE_OPTIONS}
                                selected={audioSettings.bufferSize}
                                onSelect={(val) => updateSetting('bufferSize', val)}
                                accent={COLORS.emerald + '40'}
                            />
                        </View>
                    </View>

                    {/* Sample Rate */}
                    <View style={[styles.engineCard, RETRO_SHADOW_SM]}>
                        <View style={styles.engineCardHeader}>
                            <View style={[styles.settingIcon, { backgroundColor: COLORS.secondary + '20' }]}>
                                <MaterialIcons name="graphic-eq" size={20} color={COLORS.secondary} />
                            </View>
                            <View style={{ flex: 1, marginLeft: SPACING.md }}>
                                <Text style={styles.engineLabel}>Sample Rate</Text>
                                <Text style={styles.engineDesc}>
                                    {audioSettings.sampleRate === 'auto' ? 'Hệ thống tự chọn' :
                                        audioSettings.sampleRate === '96000' ? 'Hi-Res Audio' :
                                            `${parseInt(audioSettings.sampleRate) / 1000} kHz`}
                                </Text>
                            </View>
                            <InfoButton onPress={() => showTooltip(
                                'Sample Rate',
                                '• Auto: Hệ thống Android tự chọn sample rate tối ưu cho thiết bị.\n\n• 44.1 kHz: Tiêu chuẩn CD. Phù hợp nhạc streaming thông thường.\n\n• 48 kHz: Tiêu chuẩn video/phim. Chất lượng tốt hơn 44.1 kHz.\n\n• 96 kHz (Hi-Res): Chất lượng cao nhất. Yêu cầu DAC hỗ trợ và tai nghe chất lượng. Tăng sử dụng CPU/pin.'
                            )} />
                        </View>
                        <View style={{ marginTop: SPACING.md }}>
                            <SegmentedControl
                                options={SAMPLE_RATE_OPTIONS}
                                selected={audioSettings.sampleRate}
                                onSelect={(val) => updateSetting('sampleRate', val)}
                                accent={COLORS.secondary + '35'}
                            />
                        </View>
                    </View>
                </View>

                {/* Version */}
                <Text style={styles.version}>
                    SOM • Version {Constants.expoConfig?.version || "1.0.10"}
                </Text>
            </ScrollView>

            {/* ═══ Tooltip Modal ═══ */}
            <Modal visible={tooltipVisible} transparent animationType="fade">
                <TouchableWithoutFeedback onPress={() => setTooltipVisible(false)}>
                    <View style={styles.modalOverlay}>
                        <TouchableWithoutFeedback>
                            <View style={[styles.tooltipCard, RETRO_SHADOW_SM]}>
                                <View style={styles.tooltipHeader}>
                                    <MaterialIcons name="info" size={22} color={COLORS.secondary} />
                                    <Text style={styles.tooltipTitle}>{tooltipContent.title}</Text>
                                    <TouchableOpacity onPress={() => setTooltipVisible(false)}>
                                        <MaterialIcons name="close" size={22} color={COLORS.textMuted} />
                                    </TouchableOpacity>
                                </View>
                                <View style={styles.tooltipDivider} />
                                <Text style={styles.tooltipBody}>{tooltipContent.body}</Text>
                            </View>
                        </TouchableWithoutFeedback>
                    </View>
                </TouchableWithoutFeedback>
            </Modal>
        </View>
    );
};

// ─── Styles ────────────────────────────────────────────────────────
const styles = StyleSheet.create({
    container: { flex: 1, backgroundColor: COLORS.background },

    // Profile
    profileSection: { paddingHorizontal: SPACING.lg, marginTop: SPACING.md },
    profileCard: {
        flexDirection: 'row', alignItems: 'center',
        backgroundColor: COLORS.card, padding: SPACING.md,
    },
    profileAvatar: {
        width: 52, height: 52, borderRadius: 26,
        borderWidth: 2, borderColor: COLORS.border,
    },
    profileInfo: { flex: 1, marginLeft: SPACING.md },
    profileName: { fontSize: FONT_SIZE.lg, fontWeight: '800', color: COLORS.textDark },
    badgeRow: { flexDirection: 'row', marginTop: 4 },
    badge: {
        flexDirection: 'row', alignItems: 'center', gap: 3,
        backgroundColor: COLORS.primary + '20', paddingHorizontal: 8, paddingVertical: 2,
        borderRadius: RADIUS.full,
    },
    badgeText: { fontSize: 10, fontWeight: '700', color: COLORS.primary },

    // Section
    sectionHeader: {
        flexDirection: 'row', alignItems: 'center',
        paddingHorizontal: SPACING.lg, marginTop: SPACING.xl, marginBottom: SPACING.sm,
        gap: SPACING.sm,
    },
    sectionTitle: {
        fontSize: FONT_SIZE.xs, fontWeight: '700', color: COLORS.textMuted, letterSpacing: 1.5,
    },
    sectionBadge: {
        flexDirection: 'row', alignItems: 'center', gap: 3,
        backgroundColor: COLORS.secondary + '15', paddingHorizontal: 8, paddingVertical: 2,
        borderRadius: RADIUS.full, borderWidth: 1, borderColor: COLORS.secondary + '30',
    },
    sectionBadgeText: { fontSize: 9, fontWeight: '800', color: COLORS.secondary },
    section: { paddingHorizontal: SPACING.lg },

    // Setting Row
    settingRow: {
        flexDirection: 'row', alignItems: 'center',
        backgroundColor: COLORS.card,
        borderWidth: 2, borderColor: COLORS.border,
        borderRadius: RADIUS.sm,
        padding: SPACING.md, marginBottom: SPACING.sm,
    },
    settingIcon: {
        width: 38, height: 38, borderRadius: RADIUS.sm,
        justifyContent: 'center', alignItems: 'center',
    },
    settingInfo: { flex: 1, marginLeft: SPACING.md },
    settingLabel: { fontSize: FONT_SIZE.md, fontWeight: '700', color: COLORS.textDark },
    settingSubtitle: { fontSize: FONT_SIZE.xs, color: COLORS.textMuted, marginTop: 2 },

    // Audio Engine Card
    engineCard: {
        backgroundColor: COLORS.card,
        borderWidth: 2, borderColor: COLORS.border,
        borderRadius: RADIUS.sm,
        padding: SPACING.md,
        marginBottom: SPACING.md,
    },
    engineCardHeader: {
        flexDirection: 'row', alignItems: 'center',
    },
    engineLabel: { fontSize: FONT_SIZE.md, fontWeight: '800', color: COLORS.textDark },
    engineDesc: { fontSize: FONT_SIZE.xs, color: COLORS.textMuted, marginTop: 2 },

    // Info Button
    infoBtn: {
        width: 28, height: 28, borderRadius: 14,
        backgroundColor: COLORS.background,
        borderWidth: 1.5, borderColor: COLORS.border,
        justifyContent: 'center', alignItems: 'center',
    },

    // Tooltip Modal
    modalOverlay: {
        flex: 1, backgroundColor: 'rgba(0,0,0,0.35)',
        justifyContent: 'center', alignItems: 'center',
        paddingHorizontal: SPACING.xl,
    },
    tooltipCard: {
        backgroundColor: COLORS.card,
        borderWidth: 2, borderColor: COLORS.border,
        borderRadius: RADIUS.sm,
        padding: SPACING.lg,
        width: '100%',
        maxWidth: 360,
    },
    tooltipHeader: {
        flexDirection: 'row', alignItems: 'center', gap: SPACING.sm,
    },
    tooltipTitle: {
        flex: 1, fontSize: FONT_SIZE.lg, fontWeight: '800', color: COLORS.textDark,
    },
    tooltipDivider: {
        height: 2, backgroundColor: COLORS.border, marginVertical: SPACING.md,
    },
    tooltipBody: {
        fontSize: FONT_SIZE.sm, color: COLORS.textDark, lineHeight: 20, fontWeight: '500',
    },

    // Version
    version: {
        textAlign: 'center', color: COLORS.textMuted,
        fontSize: FONT_SIZE.xs, marginTop: SPACING.xl, fontWeight: '600',
    },
});

export default SettingsScreen;
