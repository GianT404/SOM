import React from 'react';
import { View, StyleSheet, StyleProp, ViewStyle } from 'react-native';

export interface NeoShadowWrapperProps {
    children: React.ReactNode;
    style?: StyleProp<ViewStyle>;
    containerStyle?: StyleProp<ViewStyle>;
    borderRadius?: number;
    offset?: number;
}

export const NeoShadowWrapper: React.FC<NeoShadowWrapperProps> = ({
    children,
    style,
    borderRadius = 0,
    offset = 4,
    containerStyle
}) => {
    return (
        <View style={[{ position: 'relative' }, containerStyle]}>
            <View style={[
                StyleSheet.absoluteFillObject,
                {
                    backgroundColor: '#1A1A1A',
                    borderRadius: borderRadius,
                    transform: [{ translateX: offset }, { translateY: offset }]
                }
            ]} />
            <View style={[
                {
                    borderWidth: 2,
                    borderColor: '#1A1A1A',
                    borderRadius: borderRadius,
                    overflow: 'hidden',
                    backgroundColor: '#FFF'
                },
                style
            ]}>
                {children}
            </View>
        </View>
    );
};