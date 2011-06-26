# trashmail

## Why?!?

I was tired of unsubscribing from newsletters and so on, and didn't feel comfortable using public trashmail services. Also, my approach has the benefit that you are able to receive emails from them again whenever you feel like it. This is how it works:

You add a line to your ~/.maillist file that looks like this (without the quotes, without spaces, but with a \t in between): 'random@emailaddress [tab] servicename'. Once added, you can go to a service like facebook, create an account, and fetch the verification email. If you don't want to receive more emails from them for now, just replace the '@' from the email address  with a '!', and you won't be bothered again by emails from them. Once you'd like to receive emails from them again (because you lost your password, or you just feel like it), just place the '@' back in it again.

Emails you receive have a [servicename] tag added to the subject line, so you can quickly recognize who sent it to you.

## Setup

I have created an example .procmailrc, which you should place in your home folder. You need to change the address your emails are forwarded to in the last line, or, if you run your own imap / pop3 server, you need to set the folder procmail should move emails to.

Then we have the .trashmailrc, which needs to be placed in your $HOME aswell. Set the domain you use for trash mailing, choose an emailaccount for the "To: " field (used for easier filtering), and set where trashmail can find your .maillist file.

Remember to setup a catchall emailaddress with your MTA.

Please note that any email that doesn't match an entry in the list will be silently discarded. This is intentional.

## Limitations

Due to the currently (r57) extremely limited regexp capabilities of go, email addresses you add to your .maillist can only contain alphabetic characters. This could be extended to also allow digits. Any input on making the regexps more robust is welcome!
