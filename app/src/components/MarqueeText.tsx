import React, { useEffect, useState, useRef } from 'react';
import { View, Text, Animated, Easing, StyleSheet, StyleProp, TextStyle, ScrollView } from 'react-native';

interface MarqueeTextProps {
    text: string;
    style?: StyleProp<TextStyle>;
    speed?: number; // pixels per millisecond (default: 0.05 = ~50px/s)
}

const MarqueeText: React.FC<MarqueeTextProps> = ({ text, style, speed = 0.05 }) => {
    const [containerWidth, setContainerWidth] = useState(0);
    const [textWidth, setTextWidth] = useState(0);
    const translateX = useRef(new Animated.Value(0)).current;
    const animRef = useRef<Animated.CompositeAnimation | null>(null);

    const shouldScroll = textWidth > 0 && containerWidth > 0 && textWidth > containerWidth;

    useEffect(() => {
        setTextWidth(0);
    }, [text]);

    useEffect(() => {
        // Dừng animation cũ
        animRef.current?.stop();
        animRef.current = null;
        translateX.setValue(0);

        if (!shouldScroll) return;

        // Gap giữa 2 bản text bằng đúng containerWidth → seamless loop
        const loopDistance = textWidth + containerWidth;
        const duration = loopDistance / speed;

        let isCancelled = false;

        const runLoop = () => {
            if (isCancelled) return;
            translateX.setValue(0);
            const anim = Animated.timing(translateX, {
                toValue: -loopDistance,
                duration,
                easing: Easing.linear,
                useNativeDriver: true,
            });
            animRef.current = anim;
            anim.start(({ finished }) => {
                if (finished && !isCancelled) {
                    runLoop();
                }
            });
        };

        // Delay nhỏ để đảm bảo layout ổn định
        const timer = setTimeout(runLoop, 80);

        return () => {
            isCancelled = true;
            clearTimeout(timer);
            animRef.current?.stop();
            animRef.current = null;
        };
    }, [shouldScroll, textWidth, containerWidth, text]);

    return (
        <View
            style={styles.marqueeContainer}
            onLayout={(e) => setContainerWidth(Math.floor(e.nativeEvent.layout.width))}
        >
            <ScrollView
                horizontal
                showsHorizontalScrollIndicator={false}
                scrollEnabled={false}
                bounces={false}
            >
                <Animated.View
                    style={[styles.animatedRow, { transform: [{ translateX }] }]}
                >
                    {/* Text gốc */}
                    <Text
                        onLayout={(e) => setTextWidth(Math.floor(e.nativeEvent.layout.width))}
                        style={[style, styles.textBase, !shouldScroll && { textAlign: 'center', width: containerWidth || '100%' }]}
                        numberOfLines={1}
                    >
                        {text}
                    </Text>

                    {/*
                     * Clone text dùng marginLeft thay vì position: 'absolute'
                     * → nằm trong normal flow, không bị clip bởi Animated.View
                     * → khoảng cách gap = containerWidth để loop liền mạch
                     */}
                    {shouldScroll && (
                        <Text
                            style={[style, styles.textBase, { marginLeft: containerWidth }]}
                            numberOfLines={1}
                        >
                            {text}
                        </Text>
                    )}
                </Animated.View>
            </ScrollView>
        </View>
    );
};

const styles = StyleSheet.create({
    marqueeContainer: {
        overflow: 'hidden',
        width: '100%',
        justifyContent: 'center',
    },
    animatedRow: {
        flexDirection: 'row',
        alignItems: 'center',
    },
    textBase: {
        flexShrink: 0,   // Không bị nén trong flex row
        flexWrap: 'nowrap',
    },
});

export default MarqueeText;