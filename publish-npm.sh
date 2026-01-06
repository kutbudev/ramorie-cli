#!/bin/bash

echo "🚀 npm Publish Script"
echo "====================="
echo ""
echo "Authenticator app'inden 6 haneli 2FA kodunu gir:"
read -p "OTP Code: " OTP_CODE

cd npm
echo ""
echo "📦 Publishing ramorie@2.4.0 to npm..."
npm publish --access public --otp=$OTP_CODE

if [ $? -eq 0 ]; then
    echo ""
    echo "✅ Successfully published to npm!"
    echo "🔗 https://www.npmjs.com/package/ramorie"
else
    echo ""
    echo "❌ Publish failed. Check the error above."
fi
