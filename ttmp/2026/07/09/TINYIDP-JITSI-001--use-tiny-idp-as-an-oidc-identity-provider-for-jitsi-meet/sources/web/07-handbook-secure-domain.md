> [!-secondary] -secondary
> note
> 
> This method of authentication is deprecated and should not be used in new installations. It is recommended to use JWT authentication instead, as described in [Authentication](https://jitsi.github.io/handbook/docs/devops-guide/authentication).

It is possible to allow only authenticated users to create new conference rooms. Whenever a new room is about to be created, Jitsi Meet will prompt for a user name and password. After the room is created, others will be able to join from anonymous domain. Here's what has to be configured:

## Prosody configuration

If you have installed Jitsi Meet from the Debian package, these changes should be made in `/etc/prosody/conf.avail/[your-hostname].cfg.lua`

In the example below, this hostname is assumed to be `jitsi.example.com`. Update this value according to your own hostname.

### Enable authentication

Inside the `VirtualHost "[your-hostname]"` section, replace anonymous authentication with hashed password authentication:

```markdown
VirtualHost "jitsi.example.com"
    authentication = "internal_hashed"
```

You will see your own hostname instead of `jitsi.example.com` in your config file.

### Enable anonymous login for guests

Add this section **after the previous VirtualHost** to enable the anonymous login method for guests:

```markdown
VirtualHost "guest.jitsi.example.com"
    authentication = "jitsi-anonymous"
    c2s_require_encryption = false
```

*Note that `guest.jitsi.example.com` is internal to Jitsi, and you do not need to (and should not) create a DNS record for it, or generate an SSL/TLS certificate, or do any web server configuration. While it is internal, you should still replace `jitsi.example.com` with your hostname.*

## Jitsi Meet configuration

In config.js, the `anonymousdomain` options has to be set.

If you have installed jitsi-meet from the Debian package, these changes should be made in `/etc/jitsi/meet/[your-hostname]-config.js`.

```markdown
var config = {
    hosts: {
        domain: 'jitsi.example.com',
        anonymousdomain: 'guest.jitsi.example.com',
        // ...
    },
    // ...
}
```

You will see your own hostname instead of `jitsi.example.com` in your config file. You should add only the `anonymousdomain` line. Be carefull of commas.

## Jicofo configuration

When running Jicofo, specify your main domain in an additional configuration property. Jicofo will accept conference allocation requests only from the authenticated domain. This should go as a new `authentication` section in `/etc/jitsi/jicofo/jicofo.conf`:

```markdown
jicofo {
  authentication: {
    enabled: true
    type: XMPP
    login-url: "jitsi.example.com"
  }
}
```

Replace `jitsi.example.com` with your own hostname. Don't create a new `jicofo` section. Create the `authentication` section inside the existing `jicofo` section.

## Restart the services

Restart prosody, jicofo and jitsi-videobridge2 as `root`.

```markdown
systemctl restart prosody
systemctl restart jicofo
systemctl restart jitsi-videobridge2
```

## Create users in Prosody

Finally, run `prosodyctl` to create a user in Prosody:

```markdown
sudo prosodyctl register <username> <your-hostname> <password>
```

For example:

```markdown
sudo prosodyctl register myname jitsi.example.com mypassword123
```

> [!-secondary] -secondary
> note
> 
> Docker users may require an alternate config path. Users of the official [`jitsi/prosody`](https://github.com/jitsi/docker-jitsi-meet) image should invoke `prosodyctl` as follows.
> 
> ```markdown
> prosodyctl --config /config/prosody.cfg.lua register <username> meet.jitsi <password>
> ```
> 
> Full documentation for `prosodyctl` can be found on [the official site](https://prosody.im/doc/prosodyctl).

## Remove users from Prosody

To remove an existing user:

```markdown
sudo prosodyctl unregister <username> <your-hostname>
```

For example:

```markdown
sudo prosodyctl unregister myname jitsi.example.com
```

## Optional: Jigasi configuration

### Enable Authentication

If you are using Jigasi, set it to authenticate by editing the following lines in `/etc/jitsi/jigasi/sip-communicator.properties`:

```markdown
org.jitsi.jigasi.xmpp.acc.USER_ID=SOME_USER@SOME_DOMAIN
org.jitsi.jigasi.xmpp.acc.PASS=SOME_PASS
org.jitsi.jigasi.xmpp.acc.ANONYMOUS_AUTH=false
```

Note that the password is the actual plaintext password, not a base64 encoding.

### Debugging

If you experience problems with a certificate chain, you may need to uncomment the following line, also in `sip-communicator.properties`:

```markdown
net.java.sip.communicator.service.gui.ALWAYS_TRUST_MODE_ENABLED=true
```

> [!-secondary] -secondary
> note
> 
> This should only be used for testing/debugging purposes, or in controlled environments. If you confirm that this is the problem, you should then solve it in another way (e.g. get a signed certificate for Prosody, or add the particular certificate to Jigasi’s trust store).