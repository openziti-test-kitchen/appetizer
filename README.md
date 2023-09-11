OpenZiti Demo Application.
Requirements:
Go, basic Git knowledge, basic terminal commands

Start application and go to http://localhost:18000/
Enter your email and click the button to Add to OpenZiti
Read the instructions, and click on the link to download token
After you have downloaded token use command ziti edge enroll name-of-your-token.jwt
Move the json file to the directory where you built the project
Start the server using the command ./reflect client -i name-of-your-identity.json -s reflectService