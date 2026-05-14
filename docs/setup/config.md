---
icon: material/cog
---

# System Configuration & Security

Congratulations! If you have successfully completed the Docker deployment, your QA Capsule instance is now running. However, before you can start integrating your CI/CD pipelines, you must secure the platform and configure its core settings.

This guide will walk you through the initial administrative setup, from your very first login to provisioning your engineering team.

---

## 1. The Initial Boot & First Login

QA Capsule is designed with a "Secure by Default" philosophy. There is no open registration page. Upon the very first boot of the Docker container, the Go backend automatically provisions a default **System Administrator** account inside the SQLite database.

### Accessing the Portal

1. Open your web browser and navigate to the URL where you deployed QA Capsule (e.g., `http://localhost:9000` or `https://sre.yourcompany.com`).
2. You will be greeted by the QA Capsule login screen.

### Default Credentials

Unless you modified the `docker-compose.yml` environment variables, the default credentials are:

1. **Email:** `admin@qacapsule.local` (or the email specified in your `.env` file)
2. **Password:** `admin123` (or the password specified in your `.env` file)

### The Mandatory Password Reset (Action Required)

When you log in for the first time using the default credentials, you will **not** be taken to the Dashboard. 

Instead, the system will redirect you to a red **ACTION REQUIRED** screen. This is a hardcoded security measure. The system recognizes that you are using a temporary/default password and will lock you out of the Control Plane until you create a new, strong password.

* Enter a secure password.
* Click **Update & Unlock System**.
* You will be redirected to the main Dashboard. You are now the master administrator of this instance.

---

## 2. SMTP Gateway Configuration

Because QA Capsule is an enterprise-grade tool, users cannot simply "sign up." They must be invited by an Administrator. When you create a new user, the system generates a highly secure, randomized temporary password. 

To deliver this password to the new user, QA Capsule needs to know how to send emails. You must configure the SMTP Gateway.

### Configuration Steps

1. Navigate to **System Settings** using the left-hand sidebar navigation.
2. Locate the **SMTP Gateway Configuration** panel.
3. Fill in the credentials provided by your corporate email server or a transactional email service (like SendGrid, Mailgun, Amazon SES, or Mailtrap for local testing) :

   * **SMTP Host:** The address of the mail server (e.g., `smtp.sendgrid.net`).
   * **SMTP Port:** Usually `587` (for TLS) or `465` (for SSL).
   * **SMTP Username:** Your service account username.
   * **SMTP Password:** Your service account password or API token.
   * **From Email Address:** The address that will appear in the recipient's inbox (e.g., `sre-bot@mycompany.com`).

4. Click **Save & Test Connection**. The Go backend will attempt to send a test ping to the SMTP server to verify the credentials.

*Note: If the SMTP gateway is not configured, you will still be able to create users, but you will have to manually share their temporary passwords with them via a secure channel (like 1Password or a Slack DM), which is not recommended.*

---

## 3. Network & Security Policy (Domain Fencing)

If you are deploying QA Capsule in a corporate environment, you likely want to ensure that only employees of your company can access the system. QA Capsule includes a Domain Fencing feature.

1. Still in the **System Settings** menu, locate the **Network & Security Policy** card.
2. In the **Allowed Email Domains** field, enter your company's domain (e.g., `@acme-corp.com`).
3. Click **Enforce Policy**.

**What does this do?**

Once enforced, the Identity Management system will reject any attempt to provision a user whose email does not end with `@acme-corp.com`. If an administrator tries to invite a contractor using an `@gmail.com` address, the system will throw a strict `403 Forbidden` error.

---

## 4. Identity and Access Management (IAM)

Now that your system is secure and can send emails, it is time to invite your team (**SREs**, **QA Engineers**, and **Developers**).

### Global Roles Explained

QA Capsule uses a Role-Based Access Control (RBAC) system. Every user must be assigned one of three global roles:

1. **Viewer (Developer Level):** 
   
   * Can view the Dashboard, read Incident logs, and inspect StackTraces.
   * *Cannot* provision new CI/CD endpoints, cannot edit Plugins, and cannot resolve incidents.

2. **Operator (QA/Lead Level):** 
   
   * Inherits Viewer permissions.
   * Can provision new CI/CD Webhooks for their projects.
   * Can click the "Acknowledge & Resolve" button on incidents.
   * *Cannot* access the System Settings or IAM panels.

3. **Administrator (SRE/DevOps Level):** 
   
   * Has unrestricted access to the entire platform, including user management, SMTP settings, and global Plugin configurations.

### Provisioning a New User

1. Navigate to the **Users (IAM)** tab in the sidebar.
2. Fill out the **Provision Global Identity** form:

   * **Full Name:** e.g., *Jane Doe*
   * **Email Address:** e.g., *jane.doe@acme-corp.com*
   * **Global Role:** Select from the dropdown.

3. Click **Deploy Identity**.

### The User Lifecycle

Here is exactly what happens when you click "Deploy Identity":

1. The backend creates the user in the SQLite database.
2. A temporary password (e.g., `sre_temp_9f8a7b`) is generated, hashed with bcrypt, and saved.
3. The system uses the SMTP Gateway to email Jane Doe, welcoming her to QA Capsule and providing her temporary password.
4. When Jane logs in for the first time, she will hit the exact same **ACTION REQUIRED** screen you saw in Step 1, forcing her to choose a private password.

You have now successfully configured the system! You are ready to move on to setting up your `CI/CD integrations`.