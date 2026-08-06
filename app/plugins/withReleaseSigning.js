const { withAppBuildGradle } = require('@expo/config-plugins');

/**
 * Thêm signingConfig release vào app/build.gradle, đọc thông tin keystore
 * từ biến môi trường do GitHub Actions bơm vào (xem
 * .github/workflows/release-som-mobile.yml):
 *   ANDROID_KEYSTORE_FILE, ANDROID_KEYSTORE_PASSWORD, ANDROID_KEY_ALIAS, ANDROID_KEY_PASSWORD
 *
 * Nếu thiếu env thì fallback về file release.keystore với mật khẩu mặc định
 * 'changeit' để build vẫn chạy được (phục vụ build test).
 *
 * Lưu ý: buildType DEBUG luôn giữ signingConfigs.debug để build dev/test không
 * cần keystore release.
 */
module.exports = function withReleaseSigning(config) {
  return withAppBuildGradle(config, (cfg) => {
    let gradle = cfg.modResults.contents;

    // Chèn block signingConfigs.release nếu chưa có.
    if (!gradle.includes("System.getenv('ANDROID_KEYSTORE_FILE')")) {
      gradle = gradle.replace(
        "keyPassword 'android'\n        }",
        "keyPassword 'android'\n        }\n" +
          "        release {\n" +
          "            storeFile file(System.getenv('ANDROID_KEYSTORE_FILE') ?: 'release.keystore')\n" +
          "            storePassword System.getenv('ANDROID_KEYSTORE_PASSWORD') ?: 'changeit'\n" +
          "            keyAlias System.getenv('ANDROID_KEY_ALIAS') ?: 'upload'\n" +
          "            keyPassword System.getenv('ANDROID_KEY_PASSWORD') ?: 'changeit'\n" +
          "        }"
      );
    }

    // DEBUG buildType phải dùng debug keystore (idempotent).
    gradle = gradle.replace(
      /debug \{\n            signingConfig signingConfigs\.\w+/,
      "debug {\n            signingConfig signingConfigs.debug"
    );

    // RELEASE buildType dùng keystore release. Chỉ khớp block release
    // (theo sau là dòng "def enableShrinkResources"), không đụng tới debug.
    gradle = gradle.replace(
      "signingConfig signingConfigs.debug\n" +
        "            def enableShrinkResources",
      "signingConfig signingConfigs.release\n" +
        "            def enableShrinkResources"
    );

    cfg.modResults.contents = gradle;
    return cfg;
  });
};
