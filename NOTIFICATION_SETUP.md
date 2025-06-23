# 📧📱 Remote Jobs Notification Setup Guide

Get notified instantly when new remote fresher jobs are found!

⚠️ **SECURITY NOTICE:** All examples below use placeholder values. Replace with your actual credentials when configuring.

## 🚀 Quick Start

1. **Run the enhanced scraper:**
   ```bash
   go run 8advanced_remote_scraper_with_notifications.go
   ```

2. **Configure notifications:**
   ```bash
   cp notification_config_template.json notification_config.json
   # Edit notification_config.json with your actual credentials
   ```

3. **Enable your preferred notification method**

---

## 📧 Email Notifications Setup

### Step 1: Enable 2-Factor Authentication
1. Go to [myaccount.google.com](https://myaccount.google.com)
2. Navigate to **Security** → **2-Step Verification**
3. Enable if not already activated

### Step 2: Generate App Password
1. Go to **Security** → **App passwords**
2. Select **Mail** and **Other (custom name)**
3. Name it "Remote Jobs Scraper"
4. Copy the 16-character password generated

### Step 3: Update Configuration
Edit `notification_config.json`:
```json
{
  "email": {
    "smtp_host": "smtp.gmail.com",
    "smtp_port": "587",
    "from_email": "your_actual_email@gmail.com",
    "from_password": "your_16_char_app_password",
    "to_email": "yuvrajsinghnain03@gmail.com"
  },
  "enable_email": true
}
```

**Note:** Replace with actual Gmail SMTP settings when configuring.

---

## 📱 WhatsApp Notifications Setup

### Step 1: Create Twilio Account
1. Sign up at [twilio.com](https://twilio.com) (free $15 credit)
2. Complete phone verification

### Step 2: Set up WhatsApp Sandbox
1. Go to **Console** → **Messaging** → **Try WhatsApp**
2. Note your sandbox number: `+1 415 523 8886`
3. Follow the setup instructions
4. Send the join code from your WhatsApp to the sandbox number

### Step 3: Get Credentials
1. Find your **Account SID** and **Auth Token** in Console Dashboard
2. Copy these values

### Step 4: Update Configuration
Edit `notification_config.json`:
```json
{
  "whatsapp": {
    "account_sid": "your_actual_account_sid",
    "auth_token": "your_actual_auth_token",
    "from_number": "whatsapp:+14155238886",
    "to_number": "whatsapp:+919216703705"
  },
  "enable_whatsapp": true
}
```

---

## 🎯 Example Configuration (Complete)

```json
{
  "email": {
    "smtp_host": "smtp.your-provider.com",
    "smtp_port": "XXX",
    "from_email": "your-actual-email@example.com",
    "from_password": "your-actual-app-password",
    "to_email": "yuvrajsinghnain03@gmail.com"
  },
  "whatsapp": {
    "account_sid": "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "auth_token": "your_auth_token_here",
    "from_number": "whatsapp:+14155238886",
    "to_number": "whatsapp:+919216703705"
  },
  "enable_email": true,
  "enable_whatsapp": true
}
```

---

## 🔄 Testing Notifications

1. **Update your config file** with real credentials
2. **Run the scraper:**
   ```bash
   go run 8advanced_remote_scraper_with_notifications.go
   ```
3. **Check your email and WhatsApp** for job alerts!

---

## 📊 What You'll Receive

### Email Notification Includes:
- 📈 Job count summary
- 🏢 Top job highlights with company details
- 💰 Salary information
- 🔗 Direct application links
- 📅 Scraping date and time

### WhatsApp Notification Includes:
- 🎯 Quick job count alert
- 📊 Summary statistics
- 💡 Reminder to check email for details
- 🚀 Motivational message

---

## 🛠️ Troubleshooting

### Email Issues:
- ✅ **"Invalid credentials"** → Regenerate App Password
- ✅ **"SMTP timeout"** → Check internet connection
- ✅ **"No emails received"** → Check spam folder

### WhatsApp Issues:
- ✅ **"Authentication failed"** → Verify Account SID/Auth Token
- ✅ **"Number not verified"** → Complete WhatsApp sandbox setup
- ✅ **"Rate limited"** → Wait a few minutes between tests

---

## 🔐 Security Notes

- 🔒 **Never share your App Password or Auth Token**
- 🔄 **Regenerate credentials if compromised**
- 📁 **Keep `notification_config.json` secure**
- 🚫 **Don't commit credentials to version control**

---

## 🎯 Features

✅ **Instant Notifications** - Get alerted as soon as jobs are found  
✅ **Multi-Platform** - Email + WhatsApp support  
✅ **Smart Filtering** - Only remote fresher jobs  
✅ **Rich Content** - Detailed job information  
✅ **Secure Setup** - Industry-standard authentication  
✅ **Free Tier** - Uses free Gmail and Twilio plans  

---

## 📞 Contact

- 📧 Email: yuvrajsinghnain03@gmail.com
- 📱 WhatsApp: +919216703705

**Happy job hunting! 🚀** 