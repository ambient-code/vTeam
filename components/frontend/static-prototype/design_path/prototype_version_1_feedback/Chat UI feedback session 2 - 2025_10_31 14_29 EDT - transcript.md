Oct 31, 2025

## Chat UI feedback session 2 \- Transcript

### 00:00:00

   
**Daniel Warner:** so good. Oh my gosh, I look like um in the afterlife over here.  
**Andy Braren:** speaking to us from the end. Yeah.  
**Daniel Warner:** There we go. I think that's better.  
**Andy Braren:** Nice.  
**Daniel Warner:** I've got a I've got a cool experiment going now. I'll I'll send you the video later today, Andy. Actually, I'll send it to everybody or I'll just write it up in Slack, which is like taking all this feedback and the meeting notes and having Claude code munch it into a change set and then a plan.  
**Andy Braren:** Sweet.  
**Daniel Warner:** And I haven't I haven't let it rip yet to to update the prototype because I wanted to make sure Sally got her two cents in before we do that. But um but I have high hopes.  
**Andy Braren:** Awesome.  
**Daniel Warner:** Yeah.  
**Andy Braren:** Trying to do the same and make that an easy pipeline for everyone because we're doing that for mass and stuff too. Just grab the recordings, take the feedback, turn into rapid updates. It's awesome.  
**Daniel Warner:** Yeah, that's awesome.  
   
 

### 00:01:01

   
**Gage Krumbach:** I've been doing that with the Slack MCP. I just did one now where I looked at all our history identifying the stakeholders. It just completed like an hour ago.  
**Daniel Warner:** Yeah, I gota I got to hook up that Slack MCP.  
**Gage Krumbach:** It's cool.  
**Daniel Warner:** I tried a while ago and that we didn't have permissions to do it, but Which  
**Gage Krumbach:** Yeah, you have to do it kind of key way, but do what you got to do.  
**Andy Braren:** Yeah, thanks for passing that along, Gage. I haven't tried it yet, but yeah, sounds good. It's still not official. Like, it still requires uh yeah, doing something in dev tools and using the web version.  
**Gage Krumbach:** Yeah, I talked to someone from Slack and they said the official one won't be until uh 2026, sometime mid 2026\. So,  
**Sally O'Malley:** Hey.  
**Andy Braren:** Yeah.  
**Daniel Warner:** Hey, Sally.  
**Sally O'Malley:** Hi. Um, before we get started, can I mention something? I I know.  
**Daniel Warner:** This is a very very structured meeting.  
**Sally O'Malley:** I'm like, of course, I'm going to I um I just got off a call once a month.  
   
 

### 00:02:06

   
**Sally O'Malley:** I meet with Ana Sailor. She is in IBM. Uh she's a distinguished engineer in IBM, but we just met at at Summit one year.  
**Dana Gutride:** We're not  
**Sally O'Malley:** And so, um Dana, this is going to be pertinent to you, too. I just got off a call with my friend, her name's Ana Sailor, and uh she's a distinguished engineer at IBM and uh she works on all things agents and policy and Oscale and um security. And anyways, um I was showing her what I've been working on and she was blown away and she made me stop so that she could record it and I'm hoping this is okay with you.  
**Daniel Warner:** Beautiful.  
**Sally O'Malley:** Is it? And then she wants to show it to her team and she wants them to start using it. This this just just happened. I did not mean for this to happen. I just have to and I told her that we only use it for open source. It's so early. I'm like we haven't tested how good the code is or anything like so yeah that happened.  
   
 

### 00:03:04

   
**Sally O'Malley:** Just wanted to let you know.  
**Dana Gutride:** All right.  
**Sally O'Malley:** I I told her like Yeah. Yeah.  
**Dana Gutride:** Yeah, I mean IBM has I I know their orchestrate solution is not coming over with Watson X from what I've heard. So it's not clear and Daniel maybe you've actually Bill and Daniel maybe you've heard more. So like the agentic orchestration that IBM has I haven't heard any information on that that coming over as part of the uh you know what's coming back.  
**Daniel Warner:** Yeah. No, I asked about that explicitly and um Watson the the orchestrate project is going to be separate from the other pieces that they're trying to integrate now with the like document ingestion and all that. Um I can't remember what what the name of that Watsonx platform is. Might just be Watsonx. Um that's separate from orchestrate. At least that's what Danielle said and what Peter said.  
**Dana Gutride:** Yeah.  
**Bill Murdock:** No, Watson X is the name for everything together like all the Watsonx products. So, it couldn't be Watson X, but I don't know what you're talking about either.  
   
 

### 00:04:06

   
**Daniel Warner:** Orchestrate not coming over.  
**Bill Murdock:** Sorry.  
**Daniel Warner:** Yeah.  
**Bill Murdock:** Orchestra is not coming over, but I think lots of other Watsonx stuff isn't coming over either.  
**Daniel Warner:** Yeah.  
**Sally O'Malley:** What?  
**Bill Murdock:** Like Watsonx data, Watsonx gov. I don't think it's coming over. As far as I know, it's just Watsonx.ai.  
**Sally O'Malley:** Do you want me to have her reach out to you before she shows it to her team?  
**Dana Gutride:** I don't care. I mean, we just can't make it available right now because of like all the cloud stuff, right? Like, we're just we're too dependent on I think I think we would love to have other people pick it up, right?  
**Sally O'Malley:** That's what I told I told her. Yeah. And but like I think she was thinking they would just run it themselves. I mean it is an open-source project right.  
**Dana Gutride:** This this is a weird time because if we move this like I I mean Jeremy and I like we'll probably talk, but there's this thing that happens when people get excited about something and you can't stop it when it becomes a movement.  
   
 

### 00:04:58

   
**Sally O'Malley:** Okay. Okay.  
**Dana Gutride:** So, I'm not going to get in the way and I think we would be it would be a mistake to stop it, but like trying to control it is really difficult, but we need to make sure that we don't lose people at the same time, which is why I've been like on this discipline kick with with us.  
**Sally O'Malley:** Yeah.  
**Dana Gutride:** So, yeah, I don't want to get in the way, right?  
**Sally O'Malley:** Mhm.  
**Dana Gutride:** This is why we're trying to do it in the open, but we've got to figure out like like let's get that that first five minutes of using this to be like heavenly, right? And then Yeah.  
**Sally O'Malley:** Yep. I told Yeah. I told her. Yeah. Yeah. All right.  
**Dana Gutride:** But yeah, that's cool.  
**Sally O'Malley:** Cool. I I think it's it's totally fine.  
**Daniel Warner:** All right.  
**Sally O'Malley:** Yep.  
**Daniel Warner:** I actually have a I have a related question for you then, Dana, which is so I took the the feedback that we that I've gotten on this prototype from Slack and from the session yesterday and sort of used Claude code to munchge it into a plan.  
   
 

### 00:05:45

   
**Daniel Warner:** Um, and I'm going to just run the plan. Um, and I'm recording my progress as I go because I thought that it would be of interest to several groups. Is it is it a problem for me to because I go through some of the functionality in this in on ambient code platform. Is it a problem if I post that to a large group like in UX or should I No. Okay.  
**Dana Gutride:** No, we we are like we want to lead or at least facilitate the way where other people can use this as much as possible, you know. Yeah. I I think that's this is like our goal is not to build like not to sell something that everyone loves, right? Our goal is to change the way people work in Red Hat, right? And and if people are excited about what we're doing, then that's going to help us align on what that looks like. Right? This is just a means like a fun way of getting there.  
**Sally O'Malley:** She was Yeah, she was blown away by things I didn't even think she like I didn't I as she was getting excited about I was like wow this is really cool like all of the git stuff all of the git integration and like it's just uh we've yeah we have a lot going on in this project but um yes  
   
 

### 00:06:53

   
**Dana Gutride:** Yeah. Yeah. Git is like that's why I Jira like Yeah, we'll get there. I don't I don't personally care about Jira, but but having Git as a source source of truth is going to be the thing that keeps developers connected.  
**Sally O'Malley:** do you see my checklist in Jira. Did you look at it?  
**Dana Gutride:** Yeah, I saw you. I s  
**Sally O'Malley:** It's amazing. But I did fail gave G after this meeting. We'll let's stay on and like get your PR merge.  
**Gage Krumbach:** Yeah.  
**Sally O'Malley:** All right.  
**Bill Murdock:** I I mean get stuff has pros and cons though because like it it's really nice if you're just using clawed code to be able to like make a change, test it, make a change, test it, make a change,  
**Daniel Warner:** All right. So,  
**Bill Murdock:** test it, make a change, test it. If you have to push it to get and pull it from git every time, that's that's friction.  
**Dana Gutride:** Oh yeah, I agree. I but I think the friction is the same friction that git allows for with a team, right?  
   
 

### 00:07:50

   
**Dana Gutride:** As soon as you need to collaborate with other people, you need get in the room, right? If you're but but when you don't build like there's no friction requirement. If you just want to use spec locally, then then that's fine. But I I think like it it allows however engaged as you need to be from like single all the way through. And I don't think we need to be prescriptive in that. I think teams will pick it up and use it the way they they organize.  
**Bill Murdock:** Yeah, I'm just saying that it'd be nice if there was a way to be have an integrated both where you can you can iterate locally as much as you want and then automatically push it to get with one click or something like that.  
**Dana Gutride:** Yeah, I think a Yeah, but that makes that's not that's not going to be hard to do.  
**Bill Murdock:** Um, I don't know how we get there yet.  
**Dana Gutride:** Just the same way like sometimes you do a git and init locally and you start with your own git repo and then you don't even have a repository on GitHub yet until you're ready or sometimes you start with a repo and then like I think that that flow will work for us.  
   
 

### 00:08:43

   
**Dana Gutride:** We're not going to be Yeah, we'll work through that.  
**Bill Murdock:** Yeah, but I think it's a little hard because like Ambient Code Platform is running out on a cloud somewhere and it's got to push all the stuff back to your local hard drive, I guess, to um Yeah, sounds  
**Daniel Warner:** Cool.  
**Dana Gutride:** I think that's that's a good one to work on.  
**Bill Murdock:** good.  
**Sally O'Malley:** I don't know.  
**Daniel Warner:** So, I will just I'll walk through the flow. Um, and you guys can just offer up feedback. I don't know, Bill and Sally if you guys if you watch the video, but I think it's helpful.  
**Bill Murdock:** I I watched I watched the video but I did not watch the recording of yesterday's session.  
**Daniel Warner:** Yeah.  
**Bill Murdock:** I didn't have time for that. Sorry.  
**Daniel Warner:** Okay. I'll I'll point out what's going to change. I'm I'm doing the thing that irritates me, which is presenting designs where we already have feedback on it, but I wanted before I made any changes, I wanted to make sure everybody felt like, you know, everyone got ample  
   
 

### 00:09:19

   
**Bill Murdock:** Sounds good.  
**Daniel Warner:** opportunity to chime in. Yep.  
**Bill Murdock:** Yeah, I felt like I had a bunch of changes, but hopefully they're already in. So, so you'll go over them as you go. That sounds  
**Daniel Warner:** All right. Cool. So, um, so this is the projects page. Um, not much has changed on this page. uh reorganize a things a little or not reorganize but just clean things up a little bit, added in some dividers, visual changes, that type of stuff. But it's the same just like super basic theme. Um the integrations um tab or navigation item I pulled out of the the user menu, but Gage has already had some feedback on that um that that it doesn't really make sense for it to be there. we got to move it back into the the user menu to sort of like reinforce the idea that it's like scoped to users. Um the action items I just brought them down a little bit closer to the table that they modify um as opposed to having them up here in sort of like more of a global space.  
   
 

### 00:10:24

   
**Daniel Warner:** Um, and the I guess the the most major change here is I brought the running status for projects that have running sessions in them um up to the projects page just so that it could function a little bit more like a dashboard and give people an insight into like what they may more urgently need to look at because it's it's running or or it's in process. So don't even know if that's possible, but I think it would be helpful um in just in terms of like directing attention. So in this case like uh you can create a new project. Um one small usability upgrade but I think an important one is just putting in default values for the project name and and display name. Um this is like this is just for early users for ease of onboarding that type of stuff. I watched two sessions in a row where the person like had to take multiple runs at this forum in order to get it going. So I figure two clicks to create a project and we're off to the races.  
   
 

### 00:11:22

   
**Daniel Warner:** Um, so instead of taking you directly to the project page, which is what happens now in the UI, we stay on this page. Um, and and the new project is is added to the table. Um, I did that because for the same reason those same two sessions there, the people were like scrolling around. They were trying to parse sort of like where they were. um it wasn't evident to them that they had entered into a project um or that they had entered into a session for that matter. So um just trying to help the user keep context and keep them oriented. So create a new project and then if you want to go and look at a project you just click on it same as you do now.  
**Bill Murdock:** Do we really need Sorry to interrupt, but do we really need both display name and project name?  
**Daniel Warner:** Um, so yeah, it's a pattern that ex  
**Bill Murdock:** I'm not sure I understand why we have one of each.  
**Gage Krumbach:** It's a Kubernetes thing. So if you people want to see the Kubernetes name, it's the ID.  
   
 

### 00:12:17

   
**Gage Krumbach:** So you can have things that have the same display name. So it's good to have the ID.  
**Bill Murdock:** Okay, I guess fair.  
**Daniel Warner:** that exists on everything that I've worked so far at Red Hat, which I've not been here long, but the projects that I've been on all involve that. And I assume that it is just what Gage said, which is like it's specific to like Kubernetes-based projects and probably has like value in terms of like the admin or setup or whatever. Um anyway, so now you're on the project page and not a ton has changed here. um the the sessions have been sort of like put front and center um because from seeing the the onboarding that people are going through like we're really trying to drive them to a session. So I wanted to make that more prominent in the project view. So here we are on the projects view and the sessions are front and center. Um, now you know, Gage's initial feedback, I know you kind of walked it back, but I I think it I think your first impression was right, which is like it's not immediately apparent here that I'm at the project level.  
   
 

### 00:13:25

   
**Daniel Warner:** So, I think um what I'm going to do is and like Jeremy had suggested in Slack to um just ditch this product in project information tab and I'm just going to bring some of this information into the project header to make it make it sort of like really say okay you're in a project. um the workflows tab. Uh Dana suggested yesterday just pulling this out um because it sort of it creates sort of a second flow for getting into a session and it just feels extraneous at this point. So I think to clean things up, we'll probably remove the workflows tab as well.  
**Bill Murdock:** So what hap So how do you get into a a session?  
**Daniel Warner:** Uh we'll get into that.  
**Bill Murdock:** You just go into sessions. Okay, cool. Sorry.  
**Daniel Warner:** No, no, no. I I'll get there. I'm just walking through this Right.  
**Bill Murdock:** This is just a different way to get into sessions is going away. It's not you're going to get there. Okay, cool.  
   
 

### 00:14:16

   
**Daniel Warner:** Right. Um and so API keys, uh this caused a little bit of confusion with some of the early users and yesterday, um I can't remember who suggested it, but basically saying like until we have like a full-fledged Jira integration or some other thirdparty integration, we don't really need this API key se uh section. So we can like put it to the side and then once we have something to offer people like okay give Jira you know an API key so that it can access this workflow um then then we'll add it back in. Yeah, Bill.  
**Bill Murdock:** Uh, I guess I was just confused by what it's trying to do.  
**Daniel Warner:** Yeah, Andy.  
**Andy Braren:** Yeah, one clarification work on your workflows point. I remember that discussion from yesterday. I think Danny, your point, we also discussed like moving that workflows area or something and making it like a workflow management area potentially. Um maybe it's within project settings or whatever, but like having some place to manage and add or whatever the workflows that are available to you inside this particular project is probably still a need, but the need for initiation  
   
 

### 00:15:09

   
**Dana Gutride:** Yeah, I think so.  
**Andy Braren:** from that workflows area probably isn't needed. I think that's what we Yeah.  
**Daniel Warner:** Okay.  
**Bill Murdock:** Do we really need separate concept of workflow in session? I don't know. Maybe. Yeah, I guess.  
**Daniel Warner:** Yeah.  
**Dana Gutride:** Yeah, these are like the core things other teams will bring to do their to do their type of work, which we're hearing a lot of request for those already.  
**Gage Krumbach:** Yeah.  
**Sally O'Malley:** What do you think about calling it like a generic session versus a workflow? Like because a workflow is a more structured session. So, but you said you you're going to move the workflows into the sessions. You're going to walk through that part.  
**Daniel Warner:** Yeah. Yeah. I I am.  
**Sally O'Malley:** Okay.  
**Daniel Warner:** So, I'll just I'll cover one more thing and then we'll we'll get into the sessions here, which is sharing. This used to be called permissions. Now, it's called sharing. Just to try and clarify what that is.  
   
 

### 00:16:06

   
**Daniel Warner:** Um, nothing else has changed in this view. Yeah. page.  
**Gage Krumbach:** Uh, sorry it was on the workflow stuff. If we come back to it, we can talk about it.  
**Daniel Warner:** All right. So, here's where here's where you land. Sessions are front and center. Um, you can click into a session by clicking on any one of these or you can create a new session similar to the way that you can do now, except I moved the choice of creating an interactive session or a headless session into this into the select instead of the form. So if you choose interactive session, it follows the same pattern as creating a project which is um you just sort of like have reasonable defaults here and you can choose to create a session and it creates it in the table. It doesn't take you strictly straight to the session view. Um also uh I did collapse a lot of the options around like fine grain model control into like an advanced tab type of situation.  
   
 

### 00:17:01

   
**Daniel Warner:** Um, this is really just to to ease onboarding. You know, I didn't want to like remove any of the functionality. I just um the these things are like were not even addressed by the early users and they just kind of were adding noise to the view. So, I I just wrapped them up for now. So, you create a session. Uh, okay. We got hands up here. Sorry. Like my like it's really hard for me to see the chat window. So, also feel free to just cut me off and chime in, but  
**Bill Murdock:** Yeah, sounds good. Sally's first.  
**Sally O'Malley:** Okay. So, uh um what I'm what I'm worried about, first of all, like this is awesome and you're like clearly a UX expert. So, um but what I'm concerned about is that the user is going to go into the sessions and they're not going to know that there are um like that's that's not the cool thing. The cool thing is like the RF where you have your repos all laid out for you.  
   
 

### 00:17:52

   
**Sally O'Malley:** But may I I don't think I'm asking my question right, but um instead of like a new session up there, could we have like new our a new workspace um plus or a new session? Could we add a tab there for like new work workspace or workflow?  
**Daniel Warner:** um yeah the I'll I'll show you let let me show you where the workflows are in the in the structure and if and if you think that it you know it's still sort of like problematic then yeah we  
**Sally O'Malley:** Okay.  
**Daniel Warner:** can I can suggest some other other ways to to structure it but I think I think I have it pretty close and I and it it you still do get the idea of generic session and and I'll show you that in a second. Uh, Bill Yeah.  
**Bill Murdock:** Yeah, I agree with Sally, but we'll see how that turns out. But also, I just find like there's a button for interactive session and a but button for headless section. I don't understand what those are. And like as a new user, why would I understand what those are?  
   
 

### 00:18:45

   
**Sally O'Malley:** God, a lot of people don't know what headlift means. Yeah.  
**Daniel Warner:** What would your impression be, Bill, as to what these  
**Bill Murdock:** Um, my guess is that like interactive session means I can do something and a headless session is just going to happen and I have no involvement with it. But like why would I kick off a session that I don't have any involvement with and can't control and doesn't do anything with me? Uh, Gage, you're on mute.  
**Gage Krumbach:** I think headless goes away. It's still a concept that's used within workflows, but if you're going to kick off something, everything I mean, even with our new update, you start a headless session and you restart it, it turns interactive.  
**Sally O'Malley:** Sure.  
**Gage Krumbach:** I think everything will just be interactive, but have headless capabilities when within a workflow. So,  
**Daniel Warner:** Okay.  
**Bill Murdock:** Sounds reasonable to me. Dan, sounds good. Thanks. All right. Uh, let's go on. Daniel Goodmore.  
**Daniel Warner:** So, so this we enter into the interactive session and this is where things get a little bit juicier.  
   
 

### 00:19:36

   
**Sally O'Malley:** Woo\!  
**Daniel Warner:** Um, so here, you know, we have the chat interface, which everybody's familiar with that at this point. And you also have um the this is what what I've been referring to it as like the formbbased workflow where you can drive the whole thing from this this sidebar if you want. You don't have to go into the chat. Um, but they both they both stay synced. So, you know, no matter where you start, um, you can continue in the other space. Um so real quick um in the formbbased side of things like in this the order of these things might change um there was a lot of discussion yesterday about exactly when and where and how and what type of repository you need to add but in the existing UI um you really need to add a spec repository before you can do anything. So that's why I have this open by default so that you come here and you're either chatting. So like at this point you could chat um you don't have a lot available to you because you haven't selected a workflow yet.  
   
 

### 00:20:43

   
**Daniel Warner:** So you have like this is what I was thinking would be kind of like Sally's idea of a of a just a generic session, right? So you can use that you can use help um the help command uh and I mean you can give it you can give it a prompt as well. Um but you know this would enable you to sort of like chat with you know Claude plus whatever we have for our our system prompt and sort of like walk you through the process of getting set up. Um there are no agents available at this point because there's no there's no spec repository seated. um that's in the chat. So here you go. Add a we'll drive the workflow first in in the formbbased one. So add a spec repo. Um in you know in this demo this user is not has not been set up by their admin to be connected with the GitHub app or um or or or any of the other or or like have been created a project with its own permissions.  
   
 

### 00:21:43

   
**Daniel Warner:** So they need to enter their GitHub name and uh their GitHub personal access token. More to the point. Save that personal access token connects to GitHub and you can select your um your spec repository. So here we select our spec repo.  
**Sally O'Malley:** I could see I could see a little popup of like what is a spec repo and then a little description of what the spec repo is there.  
**Daniel Warner:** Definitely. I think like the education in terms of like that type of development flow is something that we're going to have to like really highlight and seed through this whole thing.  
**Bill Murdock:** Yeah, it feels like it might be better handled through the chat instead. Get the chatbot to ask for the repo and you paste it in and then the hopefully the chatbot can be a little smarter than requiring you to put at the end because it can figure out what it's missing or whatever. I don't know.  
**Daniel Warner:** Oh yeah, buckle up. Buckle up, Bill.  
**Bill Murdock:** Yeah, fair enough.  
**Daniel Warner:** I can see it.  
   
 

### 00:22:38

   
**Daniel Warner:** So in this case, you save configuration, you have your spec repo, um, but you still need to seed it. And you can see that the like the chat is keeping up with you here. So it's like providing a log of your progress, but also just as as Bill suggested, it's also like capable of walk, you know, holding your hand through the process as well.  
**Sally O'Malley:** Oh, cool.  
**Daniel Warner:** So no matter where you seed it from, um, the same thing's going to happen. So like see this repository and uh it's just brings up you know the the flat file list showing you like what's in it like the the key the key piece is like um how like to your earlier point  
**Bill Murdock:** Yeah, that's nice.  
**Daniel Warner:** um Sally like how do they know what a spec repo is and how to how to set it up and all that stuff like this just assumes that one exists which is a little bit of a gap but I think I think we we can work around that.  
**Sally O'Malley:** But um if an empty repo will turn into a spec repo if you just provide a an empty GitHub repo when you seed it that's when it becomes the spec repo.  
   
 

### 00:23:38

   
**Bill Murdock:** Yeah, it would be nice if you don't already have one if it would make one for you, but that's one more feature.  
**Daniel Warner:** Right.  
**Sally O'Malley:** That's that's what seating does though.  
**Bill Murdock:** Well, now you have to still create the repo.  
**Sally O'Malley:** Ah, gotcha. Yeah.  
**Andy Braren:** Yeah, that makes sense.  
**Daniel Warner:** Cool. So, I'm like I'm I'm kind of leaning hard now on um the the setup of this in like a kind of suggest step by step, but this is also like another potential gap. So, because now the user can go and select agents um if the if the repository is not seated, I didn't show you this, I should have. It just says like you need a seated repository before you can view your agents. And like actually here now you have agents in the chat as well. Um, so, so here you can select your agent or agents that you want to include.  
**Sally O'Malley:** Oh, nice. That's so cool.  
**Daniel Warner:** Um, I brought this up yesterday. Gage said it already works this way, which is basically like if you don't select any agents, then Claude will just pick agents for you based on like what you're asking for and and what the context is.  
   
 

### 00:24:42

   
**Daniel Warner:** So, um, I I'll just remove that for now or or I'll I'll put this as a a message more than a choice. So, okay.  
**Sally O'Malley:** Well, yeah, because that that means you're not selecting any. So, not it still feels like you're selecting something. So, I kind of like it.  
**Daniel Warner:** Okay. Okay.  
**Gage Krumbach:** If you don't select anything, it's still got the possibility of using agents. Unless if you don't select something, then we inject like a don't use any agents kind of prompt.  
**Daniel Warner:** All right.  
**Sally O'Malley:** Do we?  
**Gage Krumbach:** No, I'm Yeah, it's like selecting them says please do use them.  
**Sally O'Malley:** Oh.  
**Gage Krumbach:** Not selecting them still can use them. It's just it's a black box. Who knows what happens in there?  
**Dana Gutride:** Yeah, there this isn't they are I'm noticing with every version that comes out from cloud code they are changing how they address the discoverability and also observability of what agents are doing. So I think this is maybe fine like we just hardcode this and disable it so people say oh I guess it's going to and then they'll ask and be like yeah and we put an info button next to it say yeah it's always automatically selecting them and then it this is pretty useful for people to know that it's happening.  
   
 

### 00:25:53

   
**Dana Gutride:** So, however you want to represent this as a way of train like teaching them because they will ask like oh you know like I think it's going to be a point of contention if we don't at least hint  
**Sally O'Malley:** Oh yeah.  
**Dana Gutride:** that this is happening for them.  
**Daniel Warner:** Cool. So, you have Okay, so you have your agent selected.  
**Bill Murdock:** So, so what? Go ahead.  
**Daniel Warner:** Now, you're getting into workflows finally, right? And again, I should have showed you this when it was in its disabled state, but um if you don't have a spec repo seated, you can still look at the workflows, but they're like disabled. Um but here No,  
**Bill Murdock:** So before we move on, this create RF workflow has like first thing it does is create an RF.md and then it creates a spec MD. That seems like just the IIA here seems really weird. Is that already been changed or is is is this the current RF IIA?  
**Sally O'Malley:** That's That's Yeah, that's what it is now.  
**Bill Murdock:** No, I mean is it already changed in Daniel's design based on the feedback from yesterday?  
   
 

### 00:26:46

   
**Dana Gutride:** Yeah.  
**Daniel Warner:** no, that's some one thing that we didn't talk about which is yeah in the first step you create an RF.md. So now is your job done like or why do you have to go on to specify?  
**Sally O'Malley:** I I think that can be fixed by just renaming RF.mmd to idea MD.  
**Daniel Warner:** Yeah. Yeah.  
**Bill Murdock:** Yeah. And then and then rename this workflow to to to um specify or something like that or create specification.  
**Daniel Warner:** Right. Right.  
**Bill Murdock:** Yeah, that sounds like it would address my concern. Exactly.  
**Dana Gutride:** Yeah.  
**Daniel Warner:** It's it's a good point. So, um uh I the way I have it set up here is like you can run these commands like at any point, but um Gage pointed out yesterday that no, these steps are like dependent on each other. So you like based on your repo, if you don't have an RFMD, then you're not going to be able to run specify.  
**Sally O'Malley:** Uh the first two those are the only two that aren't dependent.  
   
 

### 00:27:42

   
**Sally O'Malley:** You can run specify without an RFD for reasons but u the other ones are dependent on it.  
**Daniel Warner:** Okay. All right. So here we go. Like run this command. And um I realized yesterday that this needs to be like a status here in order to be able to truly drive this workflow completely from the form. Then this like message here needs to become a status message creating RF.md. Um but you can see in the chat the the agents were called the RF was generated um or is being generated and Right.  
**Sally O'Malley:** Can you enable streaming or does it do that?  
**Dana Gutride:** Yeah, Daniel.  
**Sally O'Malley:** Oh, sorry.  
**Dana Gutride:** Yeah. No, no, I I just want to share something. So, so the RF this idea is a is an organizational standard we have, right? Spec is separate and and we've conflated those and maybe we shouldn't. So, so we need PMS to bring RFS in a certain format. The specification is needed for engineering after that.  
**Sally O'Malley:** Mhm.  
   
 

### 00:28:42

   
**Dana Gutride:** mix mixing those two things like and engineering is gonna probably always need some type of specification in the future and maybe we need to separate that because because like even listening to Kristoff yesterday I don't think he got it like the the value of this and he has this idea that what he's already creating for engineering is valuable um and I have I have data to show that it's not so but that's going to be like but I'm wondering that by by mixing spec kit and the RF gener generation if we've if we've if we keep confusing this because that idea phase is critical for PM.  
**Sally O'Malley:** Yeah.  
**Dana Gutride:** You don't need that as a developer.  
**Sally O'Malley:** Yes.  
**Dana Gutride:** You if you go to a spec kit yourself and you give it enough information, you're going to get a decent spec out of it.  
**Daniel Warner:** Mhm.  
**Sally O'Malley:** Mhm.  
**Daniel Warner:** Right.  
**Gage Krumbach:** Um, can I add something?  
**Daniel Warner:** So even more distinction u RF workflow  
**Dana Gutride:** Yeah.  
**Sally O'Malley:** I I feel like that that that begs for the RF to MD to mirror a Jira card.  
   
 

### 00:29:38

   
**Sally O'Malley:** Then a Jira RF the RF to MD should mirror I know we're not talking about Jira. I can see see that from your face.  
**Dana Gutride:** I hate Jira, but  
**Gage Krumbach:** So I think the ideal use the ideal flow for like Kristoff or a PM would be all they do is iterate on an idea like our very very initial demo which is like hey I made this RF and it's great and then we have a workflow that goes from start to finish all the way to task. No one even touches specify. It's just like first step in the workflow rest of spec kit. No one touches it and then it goes to a repo and then developers who don't want to work on a UI platform just go to the repository grab it and use that to develop or if they want they come to the platform and then start another workflow that's like all right start implementing off this task list or whatever else.  
**Sally O'Malley:** Ah, so so you're thinking uh the PM starts the RF. You're you're frozen for some reason.  
   
 

### 00:30:41

   
**Sally O'Malley:** Maybe it's me, but the PM um is it Lee? Oh, the PM starts an RF and then clicks a button and is like, "Okay, now just create the PC."  
**Dana Gutride:** So I  
**Andy Braren:** That's  
**Sally O'Malley:** and then tells the engineers about it. And so an engineer's job starts with taking this uh this PM PO and start like iterating on that. I think that's I mean that's exactly what you just said, but I'm going to repeat it because it's kind of cool.  
**Gage Krumbach:** Yeah, I mean, yeah, Dana is hitting the nail on the head.  
**Sally O'Malley:** Yeah.  
**Gage Krumbach:** I don't think we need to teach them spec kit. That's why they're getting confused.  
**Sally O'Malley:** No. Yeah. So they do they they they do their idea and then they have a button that says now create the PC.  
**Dana Gutride:** No. Yeah, but we need to teach our we do need to teach the engineer spec kit or whatever we land on because the specification is what's going to create enough context for us to actually have a viable code  
   
 

### 00:31:23

   
**Bill Murdock:** Makes sense. And  
**Dana Gutride:** artifact by the end of the day.  
**Gage Krumbach:** Yeah, 100% agree.  
**Sally O'Malley:** Maybe just add an option to auto auto um advance through spec kit for a PM, but it it stops along the way.  
**Dana Gutride:** Yep.  
**Sally O'Malley:** It's you have to be watching it and like it's not just like hit a button and it's done. It won't finish.  
**Daniel Warner:** I wonder.  
**Gage Krumbach:** Thanks, D.  
**Daniel Warner:** Bye.  
**Bill Murdock:** So, So  
**Sally O'Malley:** Okay.  
**Daniel Warner:** Um, I wonder al uh Sally, the person that that you talked to at IBM, did they were they familiar with Speck Kit?  
**Sally O'Malley:** They weren't, but um they totally got it.  
**Daniel Warner:** Okay.  
**Sally O'Malley:** But she's a she's an engineer. Yeah.  
**Daniel Warner:** Because the what the way I was thinking from from my perspective, not a developer, keep in mind, but like I was like, this seems like an like a great way to try and use specit whereas like setting it up for myself on my own project seemed like something I didn't really want to do um or that I would have to section off a lot of time in order to try.  
   
 

### 00:32:25

   
**Sally O'Malley:** Oh, totally. Yeah.  
**Daniel Warner:** So I think there is like a good educational piece in terms of like this is spec kit it's set up you can run it very easily from this this yeah all right so  
**Sally O'Malley:** Mhm. I mean it is just pushing a button and reading.  
**Bill Murdock:** So, did we just reach a conclusion about RFS then? Because, sorry, I got pulled away like three times. Um, where did we land on whether we're doing RF and how we're doing them?  
**Sally O'Malley:** Oops.  
**Daniel Warner:** I mean the the the concept consensus seems to be that there there's some confusion around what it means to generate an RF. So, a product manager needs to be able to generate and iterate on an RF and and so like incorporating some of this spec kit stuff is not appropriate in this case, but like the the actual tactical solution to  
**Bill Murdock:** Cut it.  
**Daniel Warner:** that, I don't know, change some of the words, change some of the display. We got to figure out something that works.  
**Bill Murdock:** So tactically you could just remove the word RF alto together like you have ID8 produce idea just like we were saying earlier but then like if you want to then sell this to the PMs as an RFP generator what do you tell them to do uh gauge  
   
 

### 00:33:45

   
**Gage Krumbach:** you tell them to just type in the chat. I think we take away all these I think there's two users. The power user can run specific commands. So we have this display um for just like emulating cloud code. But then you have the workflow runner which just says all right we're running through our steps. Hey Kristoff, you need to put something in here. And like okay I'm going to do some things. Hey Kristoff, what do you think about this? It's supposed to be they're asking them questions, follow-ups and getting them out of them. they're taking them down a journey.  
**Sally O'Malley:** Yeah.  
**Gage Krumbach:** Whereas this um is for power users. It feels more like okay, I got to run this and then there's more to spec kit. There's like clarify. It's like I got to run clarify. It's like there's a lot more to it and you can be much more powerful with it. But you're the goal is to get what you want out of the PM, not necessarily teach them to be a power user.  
   
 

### 00:34:35

   
**Bill Murdock:** No, that makes sense. But if we don't take the concept of RF out of here, are we pointing to a particular artifact and saying this is the one you should use as your RF or are we just saying here's here's a million artifacts. Go pick which one you want to use for your RF.  
**Gage Krumbach:** I I think we need to stop thinking of specific flows. It's going to be generic. This RF concept, it's just the one we're focusing on now. I was talking to Michael before and it's like we need to have I think we should start thinking of two use cases side by side to sort of keep us in this more generic loop because if we keep on cycling down just everything is just like this RF um flow it's not because another team might have a different feature flow than ours. So we need I I guess saying like we should step back and think I don't know like microcopy things like these can all change to make it more clear on what steps are what but I think like the  
   
 

### 00:35:24

   
**Sally O'Malley:** Mhm.  
**Gage Krumbach:** core foundational like how a user interacts with the workflow how a user interacts with cloud code is the important bit  
**Sally O'Malley:** Mhm.  
**Daniel Warner:** and and I'm going to just jump in real quick, Andy, before you go. Sorry. Which is that I like I think when Sally suggested that this RF look more like the thing that goes in for a Jira Strat or whatever. I think that's like that could go a long way. Not that we want to be married to Jira. doesn't have to look exactly like a Jira thing, but it just should look like an RF that you can then copy and paste and take to some other workflow. So, that could alleviate it too. Sorry, Andy.  
**Bill Murdock:** Makes sense. But it's not going to be an RFP anymore, right? Sorry, Andy. Go ahead.  
**Andy Braren:** It would be right. That's what Narf has, right? It would just be in the Jira formats.  
**Bill Murdock:** No, we specifically said it's going to be an idea file and it's going to be you can do whatever you want with it.  
   
 

### 00:36:17

   
**Bill Murdock:** Like it's not going to be a thing you have to stick into your RF description. You could stick it into an epic description or a story description or email it as a memo. It's just ideas.  
**Gage Krumbach:** Yeah, it's all artifacts at the end of the day. It's just data that the model output that we are using to develop Well,  
**Andy Braren:** Yeah.  
**Bill Murdock:** Yeah. And we could have an artifact that we call IRF if we want or we don't have to if we don't want to. And I think that was the debate I was trying to get to is are we is there going to be an artifact that we call RF or not? Um and if so, what what would it where in this flow would it arrive? And I think Gage was saying we may not even have flows like this at all.  
**Gage Krumbach:** it's it's very dependent on the team. A team might really want RFS. Like they love that idea. RFS are great for them, but then one team's like, "No, we just work on everything is a architectural diagram for us. We  
   
 

### 00:37:06

   
**Gage Krumbach:** don't even do design documents." So, it's ultimately up to how the team structures things. They might not use Jira. They might use GitHub. So they'll have different integrations. So like look at I guess like look at this order more of just how a user works through a set of flows rather than always going to be going through a certain flow because I I I think it becomes generic in the end and these things get customizable. Sorry Andy I cut you off.  
**Andy Braren:** No, good. You have a thought, Bill, before I jump in. Um, I agree.  
**Bill Murdock:** Uh, yes. So, Andy, go ahead.  
**Andy Braren:** I think there's certain microcopy that we're seeing in lefth hand side that's confusing us a little bit but I totally think that ultimately um Rory product management should own the create RF workflow and it's shared across all the PMs they're all using it and they can decide you know what the final step here should be create something that's called RFMD that's fine whatever it'll be properly formatted in Jira format whatever they decide the important thing is that create RF is the very first use case and the very first workflow that we're helping PM to sort of make and define um will take care of the back end specit infrastructure what what steps are required but  
   
 

### 00:38:15

   
**Andy Braren:** ultimately like yeah I think PM owns that workflow they can determine these uh they maybe together with with us on engineering can figure out what are the steps to get there and this interface just helps them walk through those steps by steps it's almost like a like a checklist kind of a things they have to do but the final output of which can be and probably should be for you to make it more understandable RF to MD that they can very easily just copy paste into whatever work other tools they want I guess um  
**Bill Murdock:** You're suggesting that this is something that like the user would create. The workflow is something user would create. So the like ambient code platform wouldn't have an RF creation workflow. It would have a general purpose workflow editor and some some users might create a RF workflow if they wanted to.  
**Andy Braren:** Yeah, I think that's where we're going next.  
**Bill Murdock:** That seems like  
**Andy Braren:** Yeah, it and that's I think that's part of the thing that confuses us all about this effort like kind of initially pitches is oh it's an RF builder and that's all it'll be.  
   
 

### 00:39:05

   
**Andy Braren:** No, I think our mindset's shifting toward we're making a generic platform to run uh help users and help internal teams run whatever workflows they desire. Um, augmented by by AI, augmented AI, like that's a project name, right? So, I think that's sort of where we're going and this workflow. This create RF1 is the very first stab at that.  
**Bill Murdock:** Yeah, it makes sense. And that microcopy on this one needs work, but once it gets settled out, then it will be a decent illustrative example, I guess.  
**Andy Braren:** Totally agree with that. Yeah. Yes.  
**Sally O'Malley:** Uh yeah, I think a really important thing to do is just um define or understand what the difference is between uh RF.mmd and spec. MD, not not the naming, but like what's what's the difference in those two steps? Cuz that's a bit confusing. Um, even I I mean, I know what the difference is, but it I could see where that could be really confusing because it's like uh you take the RF and then you just what further um refine it into a another a more refined RF is what a spec is.  
   
 

### 00:40:11

   
**Sally O'Malley:** But and and these are the fir these are the two phases though that we are focused on for this first pilot. Am I right? I'm right about that.  
**Daniel Warner:** Yes. And I think that's why, you know, I mean that that's what we sort of like agreed on right at the outset, which is why I sort of like cut it off there. But, you know, I think this conversation just highlights like, all right, what's what's the what's the desired end state here for a product manager? Like, what's the thing that that we want them to do? Like, do they do is this good enough that they have their their these sort of artifacts or do we need to be have something more explicit?  
**Bill Murdock:** Well, they definitely need to iterate, but this UI lets them do that too, right?  
**Daniel Warner:** It it does if you're driving the whole thing through the form like it's a little harder to iterate on the actual file. Um you probably would have to do that like on the file level which which you can do.  
   
 

### 00:41:09

   
**Daniel Warner:** So once the the files are created I just I realized in watching the video yesterday that you guys aren't seeing what I'm seeing. So when I click on these files they're opening in a in a window.  
**Bill Murdock:** another window. Okay, gotcha. Oh, and then you manual edit. I don't want to manually edit. I want to go into the session and say, "No, drop the stuff about um wagg 2.1. I  
**Daniel Warner:** Right, right, right.  
**Bill Murdock:** don't think that's appropriate for this. Um, yeah.  
**Daniel Warner:** So that that would be Yeah, that would be the chatbased workflow, right? Which is like yeah like okay I want to yeah remove these dependencies or whatever and you can iterate in that way.  
**Andy Braren:** we talked sorry I think we we talked about this a little bit yesterday too that you know the ability to view the actual artifact inside this interface may or may not be helpful.  
**Daniel Warner:** Um  
**Andy Braren:** I know it's a it's a potential rabbit hole, but I I also sort of lean toward even looking at like um I don't know what cursor and other tools or or in cloud's web interface or open AI's interface is starting to get to.  
   
 

### 00:42:08

   
**Andy Braren:** Uh there's a convention or a pattern starting to emerge where there is sort of like an artifact browser or viewer or something that like appears in the right hand side. Maybe this evolves into that where there's, you know, um after you generate artifacts, you can click on one of them, one of the things that's been generated and then see that preview or see potentially even the live PR or code or whatever files like on the right hand side potentially. Um I imagine it's getting to that point eventually, but for now, for initially, I think just being able to click on one of those artifacts and have it open somewhere is okay, but I think artifact viewers probably are in our future eventually is my guess, my hunch.  
**Bill Murdock:** Yeah, artifact yours is nice, but I think it's much more urgent to have iteration interactive iteration on the artifact.  
**Andy Braren:** that'll be supported regardless I think it's an interface but seeing the artifact together alongside you iterating on it actively I think that'd be the next thing that we support  
**Bill Murdock:** Well, yeah, that's fair.  
   
 

### 00:42:57

   
**Bill Murdock:** But and Daniel, you were saying that that this is not something supported in the short term, right? That iterating on the artifact is not something in your design.  
**Daniel Warner:** Not not not on in like a I'm editing the text type of situation. I was thinking that these like the results of the workflow would right now I just have them opening in another window, but I would say like they would open in either like the GitHub editor, right? go go to where the file lives and you can edit it there. Um because especially since you can also like add other agents to help you out or whatever.  
**Bill Murdock:** Yeah, and that's that's also useful, but it's not the same as being able to tell the bot I want something different and having it in update. Yeah, exactly.  
**Daniel Warner:** Um there  
**Bill Murdock:** Casey, can you review this and suggest any edits? Yeah, exactly.  
**Sally O'Malley:** But that that's already possible though that it does that.  
**Andy Braren:** That should  
**Sally O'Malley:** There's Yeah.  
**Bill Murdock:** But but in Daniel's design, it doesn't, right?  
   
 

### 00:43:43

   
**Sally O'Malley:** No, it does. Uh it's just the the through the session through chat you can say, "Oh, it'll it'll tell you, oh, I've created RFA. MD." And then you go to the chat and be like, "Well, I want to change it." And it'll be like, "Okay, what do you want?" It does that.  
**Bill Murdock:** and then it will save it as RFMD.  
**Sally O'Malley:** Yeah. Yeah. I mean, we you you could tell it to rename it, too.  
**Bill Murdock:** Yeah, that makes sense. And that solves the problem. I I guess I thought Daniel was telling me that it won't his design won't do that, but uh  
**Daniel Warner:** No, no. Like, so like yeah, the idea is to be able to drive like a more interactive workflow through the chat.  
**Sally O'Malley:** Yeah, as long as the session's alive and running, it will  
**Daniel Warner:** So you go through all the same steps. Um, you know, you guys you guys get it, but and then you could continue to iterate on the artifact um in the chat.  
   
 

### 00:44:31

   
**Bill Murdock:** Yeah. Okay, that's what I was asking for. Thanks,  
**Daniel Warner:** So now it's seated. So now you can create your RF.  
**Sally O'Malley:** And so did you do all of this by changing the front end and like the back end and the are pretty much Oh, cool.  
**Daniel Warner:** So that's how I started, but it was just going way too slow. So this is just static HTML and JavaScript. Um um correct.  
**Bill Murdock:** See, the responses are hardcoded, right? You're not actually calling Claude. Yeah, it makes sense.  
**Daniel Warner:** They're they're generated, but they're not I'm not calling Claude in any real way. Um so, but like that's what maybe since because honestly I wanted to get it out fast. I wanted to do a couple iterations before we meet next week and just see if like take your temperature, see like is this look like a good direction or not. But now that it seems like, you know, yeah, there's some there's some still some big issues with it, but there, you know, there's enough here that people are responding positively to that we can start to actually make real pieces.  
   
 

### 00:45:36

   
**Daniel Warner:** And in my the next prototype, I'll I'll work with what we got already.  
**Bill Murdock:** Yeah, I think it's a good incremental update. I guess I would prefer a more of a blank slate kind of start over, but um I see the downsides of blank slate start over. Like in particular, I really want something that just runs on my computer, but uh I know that's not everybody's wish.  
**Daniel Warner:** Yeah, I get that. I mean, to to your first point, when I came into this team, they were already hauling ass. So, I figured I'm not gonna, you know, I didn't want to be the one to be like, let's let's take a step back.  
**Andy Braren:** What  
**Daniel Warner:** Um, and I think that there was enough, you know, the direction was was was already there. Things were already happening. So, it just didn't feel like the right time. And I think that and I and also I did see a path from where they were to where it needed to be in order to like start onboarding users.  
   
 

### 00:46:28

   
**Daniel Warner:** So, um, that was that for this for the second piece. What was your second point, Bill?  
**Bill Murdock:** Um, I think my my point was just like I would want to do a blank slate redesign that runs on the computer and not on a server.  
**Daniel Warner:** Oh, and then running locally.  
**Bill Murdock:** Yeah.  
**Daniel Warner:** That's something that I that I did not really think through and not completely sure how that would work. gauge.  
**Gage Krumbach:** I think yeah I mean going blank slate to do that it's already built with local in mind things are it depends on the workflow you're working with and like how you want to do this eventually we will have an MCP server or a CLI or whatever you can go locally and kick off jobs from your local cloud code or your local cursor Um it it's at the end of the day this thing is a hosted like  
**Daniel Warner:** He  
**Gage Krumbach:** like agent service or workflow service where you can run through workflows you run through agents um and because PMs don't have cursor they don't want to touch cursor that yeah I mean if everybody just was a developer  
   
 

### 00:47:28

   
**Bill Murdock:** Yeah, but if you could just push all those agents in the cursor, then why do you need a hosted something or other? I don't know. I know. I maybe I think if this did this all well, I think PMs would be willing to download your server and start using it.  
**Gage Krumbach:** then yes we would probably just be working all within a GitHub repo when things would be easier, but that's just not the case. That  
**Bill Murdock:** But it's possible I'm wrong. Uh Andy  
**Andy Braren:** uh to to our conversation in Slack about privacy in that element. I think there's eventually um eventually need to have like a local environment even for PMs or something usable by other roles that might even include like personal context or things they wouldn't want to have in a posted platform. So I agree with you Bill. I feel like the local use case there is something there eventually. Um and this might be sort of a tangent but I I maybe Gage or Sally can answer this. Um, is that really a hard requirement to have a full-blown CRC uh like a Open Shift local running locally?  
   
 

### 00:48:31

   
**Andy Braren:** I mean, it can just be a few containers and whatever, right? Right now it's just the overhead's super high.  
**Sally O'Malley:** That's what I Yeah.  
**Andy Braren:** Yeah.  
**Sally O'Malley:** Yeah. We were thinking, Michael and I were talking about this yesterday. I wonder if we could um just run a pod. Uh the operator is very Kubernetes specific, but um other than that, I think we could just run things in containers and pod in a pod.  
**Gage Krumbach:** There's we we've been actively trying to get it to run Kubernetes native too.  
**Sally O'Malley:** Yeah.  
**Andy Braren:** Cuz  
**Gage Krumbach:** So kind clusters should work where like you just do kind and then a script and then it should be good and then you don't have to have that. It's a lot lighter than CRC. CRC is pretty heavy.  
**Sally O'Malley:** Yeah, the difference is yeah, CRC is Open Shift specific and so if we had things in there that like only run on Open Shift like we did um until a few days ago, then Kind wouldn't work. But now I think Kind would work and it's much easier.  
   
 

### 00:49:15

   
**Gage Krumbach:** Yeah.  
**Bill Murdock:** Yeah, it kind of sounds good for like developers developing the thing, but uh in terms of a user experience be nice just just like be in cursor or something along those lines.  
**Sally O'Malley:** Yeah.  
**Bill Murdock:** U you said something about the swarm use case. Do you mean like multiple people trying to collaborate?  
**Gage Krumbach:** Um, sorry.  
**Daniel Warner:** multiple agents.  
**Gage Krumbach:** Um, the swarm case is is probably one of the main sort of end goals for this. It's like, yes, we get the processes for closed down. Now imagine a world where you have bots scraping Jira, pulling Jiraas um and then sending them out to workflows and agents then coming back and then iterating. And then you have the same thing for UX research jobs. did the same thing for RFS being refined and then it's kind of what like people like power users that use agent today agents today at cursor or cloud code to kind of just like bounce around between projects and like answer question answer question like I'll have four projects over yeah and that's what we want like PMs to be doing is jumping it's context switching but like you're putting in it's like hey answer this question okay great  
   
 

### 00:50:13

   
**Sally O'Malley:** That's what I do. Yeah. I've got like five different projects going at the same time. Yeah.  
**Gage Krumbach:** let me go do some work for you hey answer this do some work for you. You're adding like your value is that human in the loop and that's what we're doing all these steps. Now we're just trying to bring that from developers to everybody.  
**Andy Braren:** Yeah.  
**Bill Murdock:** Yeah, but I don't see the relationship between that and having to run on a server.  
**Andy Braren:** Sorry. Go ahead, Bill.  
**Bill Murdock:** But maybe I'm missing it.  
**Gage Krumbach:** It's it the swarming aspect is like otherwise you're going to have to be running all these yourself on your own machine. like somewhere is gonna have to run that like it's I don't know. I mean I guess we could make it.  
**Bill Murdock:** for a minute, but like the computer is very light on this because all the work is being done by Clyde.  
**Gage Krumbach:** Yeah. But then there's also the collaboration aspect. It's like okay I started this and now okay well someone's going to pick it up from me.  
   
 

### 00:51:14

   
**Gage Krumbach:** Maybe we have four PMs in the room and they're all working on the same RF for a reason and they're answering questions or maybe it's like PM UX and engineer and they're all answering different questions for the same session or whatever.  
**Bill Murdock:** Yeah, that makes sense. And if it was all just the same artifact, that's fine because that's in GitHub. But if it was the same session, then then you you'd either need to put the session into GitHub, too, or you would need to um some other way of sharing sessions.  
**Gage Krumbach:** Yeah. I mean it's more of like a generic concept.  
**Bill Murdock:** Yeah, that's  
**Gage Krumbach:** It's not like not technically I don't know if it's always going to be get up or it's going to be some other means but like that concept of collaborating on some agentic flow is kind of what I'm talking about.  
**Bill Murdock:** Yeah, that makes sense too. Andy.  
**Andy Braren:** uh two things. So the first uh when we get into like collaborate uh collaboration artifacts together between different people or groups and things I think it's sort of a next step uh because yeah you could go into GitHub but there might be a need to have even collaboration with on a textbased artifact or the RF inside this tool and we'll get that as a natural you know next request from the PMS we're building this tool for.  
   
 

### 00:52:21

   
**Andy Braren:** Otherwise, their current workflow very likely for Kristoff and others will be great, just copy this, paste into Jer or paste into a Google doc where they can collaborate with other PMs and refining text and whatever. And once they do that, the fact that that's now in an external system and where you'll have control of the data or the actual original artifact, it's kind of a problem for the future steps of this this process. Um, so that's a I think it's a problem a next problem for us to solve after we nail this. Um, the second thing around the the swarms and whatnot, I agree with Bill. I feel like you could have swarm running locally or just a bunch of pods or something that spin up or containers that ultimately call different external APIs for actual work to be done. Um, yeah, tabs are something around here and we can make it easier and to to your point like there is another emerging pattern around like aentic inboxes or something or like a not that we all love inboxes but  
**Bill Murdock:** or tabs.  
   
 

### 00:52:59

   
**Bill Murdock:** You could have tabs in a interface or thing. Yeah.  
**Andy Braren:** another like list of all right you have these five different projects across different things are happening and you're needed as a human for a review on these five things. click each one very easily dive into an area, look at the artifact, provide feedback, pull back out of your inbox, go back into another one, and you make that context switching easier than even what this interface shows. So, that's another augmentation in a phase two. I kind of see um the last thing I want to say, sorry Sally, to your point about workspaces in the chat, another conversation we had yesterday that you weren't here for, but um what we described earlier about those workflows being managed by the PM team of Open Shift AI somewhere somehow um that might necessitate the need for another level in this organizational hierarchy of a workspace or a team or whatever the heck we call that thing, but there needs to be some arbback grouping of like, all right, the PM's going to edit there uh you workflow workflow RF feed-workflow MD or whatever the file format that is to to define and work in that collectively.  
   
 

### 00:54:04

   
**Andy Braren:** Um so those things be being on the maybe in our phase two or or feature enhancements.  
**Sally O'Malley:** Yeah.  
**Andy Braren:** But um yeah Daniel  
**Daniel Warner:** Um, yeah, actually I think a lot of it what I was going to say has been covered, but I guess just to put a kind of a fine point on like the multi- aent aspect of this is that you know while you you know as a developer you're working with with Claude and you're working or you're working in cursor And the cursor agent is great and it's great at managing sub agents and it's really effective and all that stuff, but you know there there there is a a point in time in the not too distant future where the a multi- aent setup is not going to be um able to be contained like just within cloud code or or um or or cursor and and I and especially not for people who aren't developers. Right. So developers will live locally I think for a lot longer than most people. But for but for for a lot of people it's like okay if I want to use like a multi- aent setup what do I have to do and rather than building out like a local sort of like desktop experience it does make sense I think in my opinion to have something that's hosted that's living on a cluster that has all the resources it needs to to spin up agents ad hoc based on on the tasks that  
   
 

### 00:55:34

   
**Daniel Warner:** you're giving it. Um but but any that that was just that was my sort of like highlevel thought like um it's going to be tough to compete with with cursor and claude at this point. Um but as things evolve I think it's going to make a lot more sense. Um, and also the and I think this was covered as well, but like the just the integrations with business systems, you know, like I I know that that is possible through like, you know, cursor MCP stuff. And I'm not sure how cloud code uses MCP because I haven't done it yet, but like um you know, having all of the the context available like Andy's always talking about of like changes that are happening in Jira or changes that are happening in other like pro project and product management stuff. Being able to just access that at con as context also really sort of like contributes to the idea of this as like a hosted solution that that lives in a cluster. And anyway, that's that's more my opinion than anything else.  
   
 

### 00:56:32

   
**Daniel Warner:** But it's I I mean I just recently switched to cloud code from cursor and that experience is unreal.  
**Bill Murdock:** Yeah, those are good points. I They don't make me happy because I want the local thing, but I I understand and I I see why the other people don't like what I like. Yeah, fair.  
**Daniel Warner:** It's it's it's so good. So like it's going to be tough to compete with that for developers in the short term. Um anyway, okay, we're just about at 3:30, so unless anybody has any parting thoughts, I think we could probably wrap it there.  
**Bill Murdock:** Thanks everybody.  
**Daniel Warner:** Thanks.  
**Sally O'Malley:** One one one thing is like how far away are we from having something that looks more like this than where we are? Is this next week we'll have it do you think? Or you know that's what I'm wondering.  
**Andy Braren:** I was going to propose it actually the same idea like can we just try to make this work next week? That'd be awesome if so.  
**Sally O'Malley:** Yeah.  
   
 

### 00:57:28

   
**Sally O'Malley:** Like Monday we're all here.  
**Andy Braren:** Yeah.  
**Sally O'Malley:** Are you all traveling Monday?  
**Gage Krumbach:** I'm traveling half Monday, but I'll work on the plane.  
**Sally O'Malley:** Yeah.  
**Daniel Warner:** So, I like I talked to to Bob a little bit about that real quick after our call yesterday and I think just based on from what he said on the call and what he said after like there's gonna we're going to be split up into like different projects and probably the project that like Andy, me, him, maybe one or two other people are working on is going to be this. Um, so I I don't know. I don't want to put words in his mouth, but that was the impression that I got.  
**Sally O'Malley:** Do you anticipate it being all front end changes?  
**Gage Krumbach:** So, so y'all see this?  
**Sally O'Malley:** Gage, you know more about like the interaction. Yeah.  
**Gage Krumbach:** I think um this is what was merged. I wasn't part of the merging, so I'm kind of just seeing this. You have a number on your name and that name goes here.  
   
 

### 00:58:22

   
**Gage Krumbach:** Um first work stream getting started. So, I think this is I believe this is more of like the UX flow kind of workstream. And then there's two other like um bring your own plugin which is Michael is on that one and some others. Daniel, you're on that one too. Um so I don't know. I I also assume we're going to be in the same room so I don't know how like cohesive these work streams are. Um, I I'm on this one. So, I plan on doing a lot of uh I probably will I want to do like generic workflow stuff. I don't know what this says, but I think I I want to get that to work because I think that opens up a lot of use cases, makes things visible. Um, but this is yeah, this is the doc. I'll share it if people want to look at it more.  
**Sally O'Malley:** Yeah. Oh, but yeah, one more qu the the question that I had. Do you anticipate like to get to what Daniel's showing, is that mostly frontend changes because Mhm.  
   
 

### 00:59:31

   
**Gage Krumbach:** Um, well, I think there's a fundamental difference where you might have noticed there's no workflow page anymore. It's there's no like session workflow page. It's all session pages. So, I think we need to like technically think through how that works. Daniel, I know you said you were um didn't didn't uh there was like a disconnect.  
**Sally O'Malley:** Yeah.  
**Gage Krumbach:** I don't think like the remote was as clear as it could have been in the old design. So, like the workflow page now is just a representation of the remote GitHub whereas the session page is a representation of the workspace. like here it's kind of merged and we need to sort of at least technically bridge that gap again. Um may like hopefully we don't have to do it visually so things just feel seamless but technically I think that's the biggest thing. Otherwise it is just a lot of uh quick fixes as well as some more fundamental changes. But yeah and that's at least from my impression of looking at the mocks and trying to translate it.  
   
 

### 01:00:36

   
**Gage Krumbach:** That's what it sounds like.  
**Daniel Warner:** Cool. Um, oh, I do have one quick question. Sorry to keep expanding this, which is like, so like I I do have a branch with this prototype on it. Should I just make a fork and then share that or do you guys want to give me push access to this thing?  
**Sally O'Malley:** Uh, you have a branch of V team.  
**Daniel Warner:** Yep. Yeah.  
**Sally O'Malley:** That's fine, right?  
**Gage Krumbach:** I'm like for this for this demo is what you're saying and you're I'm confused.  
**Daniel Warner:** Yeah.  
**Gage Krumbach:** You say you want to like merge this demo into P team.  
**Daniel Warner:** No, no, I don't need to merge it, but just to make it available to you. Like I could I could just make my own fork and then you send you a link to that or do you want me to just, you know, push a branch to the project and there?  
**Sally O'Malley:** Yeah, do the main one.  
**Gage Krumbach:** Yeah.  
**Daniel Warner:** Yeah.  
**Sally O'Malley:** I either way works. Um I don't I don't know.  
   
 

### 01:01:34

   
**Gage Krumbach:** Yeah.  
**Sally O'Malley:** Um uh yeah, I've I usually I always push a a fork. I always create a fork and and push a branch to a fork, but and and I I'm curious to know like how far Cloud would go if you gave it the HTML and was like like use this  
**Daniel Warner:** Okay.  
**Gage Krumbach:** Is this just running like mock data, Daniel?  
**Daniel Warner:** All four. Yeah. Yep. It's all It's all just a static prototype.  
**Gage Krumbach:** Okay.  
**Daniel Warner:** Yep.  
**Gage Krumbach:** Yeah. And I think like our fork branch is probably fine. Yeah.  
**Sally O'Malley:** HTML and make make this work, you know, in this codebase.  
**Daniel Warner:** Yeah, it's worth trying to shove that through a couple times and see if we could just make it happen.  
**Sally O'Malley:** I wonder how Yeah.  
**Daniel Warner:** Yeah.  
**Gage Krumbach:** I should be pretty successful.  
**Andy Braren:** Maybe we should try that next week. Yeah, if the APIs are available, we can just compare the code bases and whatever and have it just figure it out. That'd be sweet.  
   
 

### 01:02:19

   
**Sally O'Malley:** Yeah. Do you think that Opus would be is is the best model to use for coding? Is it I I Yeah.  
**Gage Krumbach:** I think I think any max model would do fine. Just the screenshots in and tell it to make a plan and go there. Go from there. Um find the diffs. It's It's I think it's more of like better prompting that would get you there than just bigger models.  
**Sally O'Malley:** Yeah, because this is a complic this is a complicated thing to orchestrate. It's it's the the codebase is complicated. There's there's Python, there's TypeScript, there's Go. And so I really I want to make this happen as fast as possible. And um maybe  
**Daniel Warner:** I I actually now that I'm thinking about it like I actually would have high hopes for that approach. I don't think we could just shove it all through at once, but if we broke it up, you know, and say like, you know, update whatever the pro project page to be like this because it's all that like um that uh I can't remember the name of the front-end framework that you used shad CN or something like that, you know, seems very straightforward and claude knows its way around it pretty possible.  
   
 

### 01:03:17

   
**Gage Krumbach:** Yeah. Yeah.  
**Sally O'Malley:** Yeah. Yeah.  
**Daniel Warner:** Yeah.  
**Gage Krumbach:** Yeah. just I I'd like to whiteboard some of the complex inner workings a bit because there's two things in there like how fast the pod spins up which is just not possible with how we do it. It kind of like leans itself leans itself towards a a constantly running cloud code session which seems cool and but it changes fundamentally things and also just like how that getit integration works because that's not fundamentally there that we would need to sort of work through just so it doesn't go off and just lose features.  
**Sally O'Malley:** Yeah.  
**Gage Krumbach:** But otherwise, I think it's a great idea.  
**Daniel Warner:** Okay.  
**Gage Krumbach:** This has been great.  
**Daniel Warner:** Yeah. Thanks for the the the great session and um I'll see you all you all next week.  
**Gage Krumbach:** Thanks for the demos. Those are awesome.  
**Daniel Warner:** Yeah, my pleasure. All right. Bye.  
**Andy Braren:** think it is.  
**Gage Krumbach:** Sally, do you want to stay on it? Okay, we'll see. If not, we'll just pick up a new one.  
**Andy Braren:** It is recording. Stop that. But yeah. Yeah. Whatever. All right.  
   
 

### Transcription ended after 01:04:53

*This editable transcript was computer generated and might contain errors. People can also change the text after it was created.*