const { withAppBuildGradle } = require('@expo/config-plugins');

/**
 * Thêm signingConfig release vào app/build.gradle, đọc thông tin keystore
 * từ biến môi trường do GitHub Actions bơm vào (xem
 * .github/workflows/release-som-mobile.yml):
 *   ANDROID_KEYSTORE_FILE, ANDROID_KEYSTORE_PASSWORD, ANDROID_KEY_ALIAS, ANDROID_KEY_PASSWORD
 *
 * Nếu thiếu env thì fallback về file release.keystore với mật khẩu mặc định
 * 'changeit' để build vẫn chạy được (phục vụ build test).
 */
module.exports = function withReleaseSigning(config) {
  return withAppBuildGradle(config, (cfg) => {
    let gradle = cfg.modResults.contents;

    // Đã chèn rồi thì thôi (idempotent).
    if (gradle.includes('signingConfig signingConfigs.release')) {
      return cfg;
    }

    gradle = gradle.replaceAll(
      "keyPassword 'android'\n        }",
      "keyPassword 'android'\n        }\n" +
        "        release {\n" +
        "            storeFile file(System.getenv('ANDROID_KEYSTORE_FILE') ?: 'release.keystore')\n" +
        "            storePassword System.getenv('ANDROID_KEYSTORE_PASSWORD') ?: 'changeit'\n" +
        "            keyAlias System.getenv('ANDROID_KEY_ALIAS') ?: 'upload'\n" +
        "            keyPassword System.getenv('ANDROID_KEY_PASSWORD') ?: 'changeit'\n" +
        "        }"
    );

    gradle = gradle.replaceAll(
      'signingConfig signingConfigs.debug',
      'signingConfig signingConfigs.release'
    );

    cfg.modResults.contents = gradle;
    return cfg;
  });
};
