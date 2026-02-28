import React, { useState } from 'react';
import {
    View, Text, ScrollView, TouchableOpacity, Switch, StyleSheet, StatusBar,
} from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';
import { COLORS, SPACING, RADIUS, FONT_SIZE, RETRO_BORDER, RETRO_SHADOW, RETRO_SHADOW_SM } from '../theme';

interface SettingRowProps {
    icon: string;
    label: string;
    subtitle?: string;
    value?: boolean;
    onToggle?: (val: boolean) => void;
    onPress?: () => void;
    iconBg?: string;
}

const SettingRow: React.FC<SettingRowProps> = ({ icon, label, subtitle, value, onToggle, onPress, iconBg }) => (
    <TouchableOpacity style={styles.settingRow} onPress={onPress} activeOpacity={onToggle ? 1 : 0.7}>
        <View style={[styles.settingIcon, { backgroundColor: iconBg || COLORS.secondary + '30' }]}>
            <MaterialIcons name={icon as any} size={20} color={COLORS.darkBorder} />
        </View>
        <View style={styles.settingInfo}>
            <Text style={styles.settingLabel}>{label}</Text>
            {subtitle && <Text style={styles.settingSubtitle}>{subtitle}</Text>}
        </View>
        {onToggle !== undefined && value !== undefined ? (
            <Switch
                value={value}
                onValueChange={onToggle}
                trackColor={{ false: '#374151', true: COLORS.secondary }}
                thumbColor={value ? COLORS.primary : COLORS.darkBorder}
            />
        ) : (
            <MaterialIcons name="chevron-right" size={22} color={COLORS.darkBorder + '80'} />
        )}
    </TouchableOpacity>
);

const SettingsScreen: React.FC = () => {
    const [notifications, setNotifications] = useState(true);
    const [darkMode, setDarkMode] = useState(true);

    return (
        <View style={styles.container}>
            <StatusBar barStyle="light-content" backgroundColor={COLORS.darkBg} />
            <ScrollView showsVerticalScrollIndicator={false}>
                {/* Header */}
                <View style={styles.header}>
                    <TouchableOpacity style={[styles.backBtn, RETRO_SHADOW]}>
                        <MaterialIcons name="arrow-back" size={22} color={COLORS.darkBg} />
                    </TouchableOpacity>
                    <Text style={styles.headerTitle}>Settings</Text>
                    <View style={{ width: 36 }} />
                </View>

                {/* Profile Card */}
                <TouchableOpacity style={[styles.profileCard, RETRO_SHADOW_SM]}>
                    <View style={styles.profileAvatar}>
                        <MaterialIcons name="person" size={28} color={COLORS.secondary} />
                    </View>
                    <View style={styles.profileInfo}>
                        <Text style={styles.profileName}>Alex Johnson</Text>
                        <Text style={styles.profileBadge}>Premium Member</Text>
                    </View>
                    <MaterialIcons name="chevron-right" size={22} color={COLORS.darkBorder + '80'} />
                </TouchableOpacity>

                {/* General */}
                <Text style={styles.sectionTitle}>GENERAL</Text>
                <View style={styles.section}>
                    <SettingRow icon="person" label="Account" iconBg={COLORS.primary + '30'} />
                    <SettingRow icon="notifications" label="Notifications" value={notifications} onToggle={setNotifications} iconBg={COLORS.secondary + '30'} />
                    <SettingRow icon="headphones" label="Audio Quality" subtitle="High (320kbps)" iconBg={COLORS.accent + '30'} />
                </View>

                {/* Appearance */}
                <Text style={styles.sectionTitle}>APPEARANCE</Text>
                <View style={styles.section}>
                    <SettingRow icon="palette" label="Theme" iconBg={COLORS.emerald + '30'} />
                    <SettingRow icon="dark-mode" label="Dark Mode" value={darkMode} onToggle={setDarkMode} iconBg={COLORS.secondary + '30'} />
                </View>

                {/* Log Out */}
                <TouchableOpacity style={[styles.logoutBtn, RETRO_SHADOW_SM]}>
                    <MaterialIcons name="logout" size={20} color={COLORS.accent} />
                    <Text style={styles.logoutText}>Log Out</Text>
                </TouchableOpacity>

                <Text style={styles.version}>Version 4.2.0 (Build 198)</Text>
                <View style={{ height: 100 }} />
            </ScrollView>
        </View>
    );
};

const styles = StyleSheet.create({
    container: { flex: 1, backgroundColor: COLORS.darkBg },
    header: {
        flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center',
        paddingHorizontal: SPACING.lg, paddingTop: SPACING.xxl, paddingBottom: SPACING.md,
    },
    backBtn: {
        width: 36, height: 36, borderRadius: 18, backgroundColor: COLORS.primary,
        borderWidth: 2, borderColor: COLORS.darkBorder, justifyContent: 'center', alignItems: 'center',
    },
    headerTitle: { fontSize: FONT_SIZE.xl, fontWeight: '700', color: COLORS.darkBorder },
    profileCard: {
        flexDirection: 'row', alignItems: 'center', marginHorizontal: SPACING.lg, marginTop: SPACING.md,
        backgroundColor: COLORS.darkCard, borderWidth: 2, borderColor: COLORS.darkBorder + '40',
        borderRadius: RADIUS.xl, padding: SPACING.md,
    },
    profileAvatar: {
        width: 48, height: 48, borderRadius: 24, backgroundColor: COLORS.secondary + '20',
        justifyContent: 'center', alignItems: 'center',
    },
    profileInfo: { flex: 1, marginLeft: SPACING.md },
    profileName: { fontSize: FONT_SIZE.lg, fontWeight: '700', color: COLORS.darkBorder },
    profileBadge: { fontSize: FONT_SIZE.xs, color: COLORS.secondary, fontWeight: '500', marginTop: 2 },
    sectionTitle: {
        fontSize: FONT_SIZE.xs, fontWeight: '700', color: COLORS.textMuted, letterSpacing: 2,
        paddingHorizontal: SPACING.lg, marginTop: SPACING.xl, marginBottom: SPACING.sm,
    },
    section: { marginHorizontal: SPACING.lg },
    settingRow: {
        flexDirection: 'row', alignItems: 'center', backgroundColor: COLORS.darkCard,
        borderWidth: 2, borderColor: COLORS.darkBorder + '30', borderRadius: RADIUS.xl,
        padding: SPACING.md, marginBottom: SPACING.sm,
    },
    settingIcon: {
        width: 40, height: 40, borderRadius: RADIUS.md, justifyContent: 'center', alignItems: 'center',
    },
    settingInfo: { flex: 1, marginLeft: SPACING.md },
    settingLabel: { fontSize: FONT_SIZE.md, fontWeight: '600', color: COLORS.darkBorder },
    settingSubtitle: { fontSize: FONT_SIZE.xs, color: COLORS.textMuted, marginTop: 2 },
    logoutBtn: {
        flexDirection: 'row', alignItems: 'center', justifyContent: 'center',
        marginHorizontal: SPACING.lg, marginTop: SPACING.xl,
        backgroundColor: COLORS.accent + '20', borderWidth: 2, borderColor: COLORS.accent + '40',
        borderRadius: RADIUS.full, padding: SPACING.md, gap: SPACING.sm,
    },
    logoutText: { fontSize: FONT_SIZE.md, fontWeight: '700', color: COLORS.accent },
    version: { textAlign: 'center', color: COLORS.secondary, fontSize: FONT_SIZE.xs, marginTop: SPACING.lg, fontWeight: '500' },
});

export default SettingsScreen;
