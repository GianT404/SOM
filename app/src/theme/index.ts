
export const COLORS = {
    background: '#FFFBF5',
    card: '#FFFFFF',
    border: '#18181b',

    primary: '#FBBF24',
    secondary: '#A78BFA',
    accent: '#F472B6',

    textDark: '#1F2937',
    textMuted: '#9CA3AF',
    textLight: '#FFFFFF',

    darkBg: '#111827',
    darkCard: '#1F2937',
    darkBorder: '#e5e7eb',

    orange: '#F97316',
    pink: '#EC4899',
    emerald: '#10B981',
    red: '#EF4444',
    pinkBg: '#FECDD3',
    emeraldBg: '#A7F3D0',
};

export const SPACING = {
    xs: 4,
    sm: 8,
    md: 16,
    lg: 24,
    xl: 32,
    xxl: 48,
};

export const RADIUS = {
    sm: 8,
    md: 12,
    lg: 16,
    xl: 20,
    xxl: 24,
    full: 9999,
};

export const FONT_SIZE = {
    xs: 11,
    sm: 13,
    md: 15,
    lg: 18,
    xl: 22,
    xxl: 28,
    hero: 34,
};

export const RETRO_SHADOW = {
    shadowColor: '#000000',
    shadowOffset: { width: 12, height: 12 },
    shadowOpacity: 1,
    shadowRadius: 0,
    elevation: 12,
};

export const RETRO_SHADOW_SM = {
    shadowColor: '#18181b',
    shadowOffset: { width: 2, height: 2 },
    shadowOpacity: 1,
    shadowRadius: 0,
    elevation: 3,
};

export const RETRO_BORDER = {
    borderWidth: 2,
    borderColor: '#18181b',
};

export const HEADER = {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingHorizontal: SPACING.lg,
    paddingTop: SPACING.xxl + 8,
    paddingBottom: SPACING.sm,
    position: 'relative',
    backgroundColor: COLORS.background,
} as const;

export const headerTitleContainer = {
    position: 'absolute',
    left: 0,
    right: 0,
    top: SPACING.xxl + 8,
    bottom: SPACING.sm,
    justifyContent: 'center',
    alignItems: 'center',
    zIndex: 0,
} as const;

export const avatar = {
    width: 44, height: 44, borderRadius: 22, ...RETRO_BORDER,
    backgroundColor: COLORS.primary + '30', justifyContent: 'center', alignItems: 'center',
} as const;

export const headerTitle = {
    fontSize: FONT_SIZE.lg,
    fontWeight: '600',
    color: COLORS.textDark,
} as const;

export const headerLeft = {
    minWidth: 44,
} as const;

export const searchBar = {
    flexDirection: 'row', alignItems: 'center', marginHorizontal: SPACING.lg,
    backgroundColor: COLORS.card, ...RETRO_BORDER, borderRadius: RADIUS.full,
    height: 48, marginBottom: SPACING.lg,
} as const;

export const searchInput = {
    flex: 1, paddingHorizontal: SPACING.lg, fontSize: FONT_SIZE.md, color: COLORS.textDark,
} as const;

export const searchIcon = {
    width: 38, height: 38, borderRadius: 19, backgroundColor: COLORS.background,
    justifyContent: 'center', alignItems: 'center', marginRight: SPACING.xs,
} as const;

export const backBtn = {
    width: 44,
    height: 44,
    backgroundColor: '#FFB26B',
    borderRadius: RADIUS.md,
    ...RETRO_BORDER,
    justifyContent: 'center',
    alignItems: 'center',
} as const;
