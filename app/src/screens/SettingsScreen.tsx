import React, { useState } from 'react';
import {
    View, Text, ScrollView, TouchableOpacity, Switch, StyleSheet, StatusBar, Image,
} from 'react-native';
import { MaterialIcons, Ionicons } from '@expo/vector-icons';
import {
    COLORS, SPACING, RADIUS, FONT_SIZE,
    HEADER, headerTitleContainer, headerTitle, headerLeft, backBtn,
} from '../theme';
import { NeoShadowWrapper } from '../components/NeoShadowWrapper';

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

const SettingsScreen: React.FC<{ navigation: any }> = ({ navigation }) => {
    const [notifications, setNotifications] = useState(true);
    const [darkMode, setDarkMode] = useState(false);
    const [autoPlay, setAutoPlay] = useState(true);

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
            </ScrollView>
        </View>
    );
};

const styles = StyleSheet.create({
    container: { flex: 1, backgroundColor: COLORS.background },

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

    sectionTitle: {
        fontSize: FONT_SIZE.xs, fontWeight: '700', color: COLORS.textMuted, letterSpacing: 1.5,
        paddingHorizontal: SPACING.lg, marginTop: SPACING.xl, marginBottom: SPACING.sm,
    },
    section: { paddingHorizontal: SPACING.lg },
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

    version: {
        textAlign: 'center', color: COLORS.textMuted,
        fontSize: FONT_SIZE.xs, marginTop: SPACING.xl, fontWeight: '600',
    },
});

export default SettingsScreen;
