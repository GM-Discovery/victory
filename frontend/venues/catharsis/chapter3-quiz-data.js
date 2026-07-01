// Chapter 3 archetype quiz data, ported verbatim from the canonical
// reference (script.js / chapter-3-archetype-quiz-v1.0.0.json): 15
// questions, 89 answers, scoring weights. This data is intentionally
// client-only per Kernel 55 section 6 — it is never sent to the server.
(function () {
  window.VictoryChapter3QuizData = {
    datasetVersion: "1.0.0",
    questions: [
      {
        text: "A wounded person approaches you. What do you do first?",
        maxSelections: 2,
        answers: [
          { text: "I help them sit or steady themselves. I can figure out the rest once they are not standing there hurt and scared.", scores: { Comforter: 7, Steward: 3, Guardian: 2, Reconciler: 1, Builder: 1 } },
          { text: "I look around before getting close. If something did this to them, I want to know whether it is still nearby.", scores: { Guardian: 7, Watcher: 4, Strategist: 3, Challenger: 1, Steward: 1 } },
          { text: "I ask what happened while checking what I can actually see. Their answer matters, but so do the details.", scores: { Watcher: 7, Seeker: 4, Strategist: 3, Challenger: 2, Architect: 1 } },
          { text: "I call for help and start giving people simple things to do. This is easier to handle if the whole moment is not sitting on one person.", scores: { Catalyst: 7, Performer: 4, Builder: 3, Reconciler: 2, Strategist: 1 } },
          { text: "I want to know what led to this. People don't usually end up hurt for no reason.", scores: { Firebrand: 7, Challenger: 5, Seeker: 3, Strategist: 2, Watcher: 1 } }
        ]
      },
      {
        text: "A friend approaches you and asks to speak privately. What is your first thought?",
        maxSelections: 2,
        answers: [
          { text: "Something is bothering them. Whatever this is, it probably matters to them.", scores: { Comforter: 8, Reconciler: 3, Steward: 2, Guardian: 1 } },
          { text: "That's unusual. I start thinking about what might have led to this conversation.", scores: { Watcher: 8, Seeker: 3, Strategist: 2, Challenger: 1 } },
          { text: "If they want privacy, I'll give it to them. Some conversations deserve space.", scores: { Guardian: 6, Steward: 4, Reconciler: 2, Comforter: 1 } },
          { text: "I wonder what they need from me specifically.", scores: { Strategist: 6, Steward: 3, Comforter: 2, Architect: 2, Builder: 1 } },
          { text: "I wonder whether this conversation is really about the thing they're about to say.", scores: { Challenger: 7, Watcher: 3, Seeker: 3, Architect: 1 } },
          { text: "This must have taken some courage to bring up.", scores: { Performer: 7, Comforter: 3, Reconciler: 2 } }
        ]
      },
      {
        text: "You come to own a parcel of land. What is your first thought?",
        maxSelections: 2,
        answers: [
          { text: "I start imagining what could be built there. Empty space is full of possibilities.", scores: { Builder: 7, Architect: 4, Strategist: 2, Catalyst: 2 } },
          { text: "I want to clean it up and restore it. Whatever happens next, it should be cared for first.", scores: { Aesthetician: 7, Steward: 4, Builder: 2, Comforter: 1 } },
          { text: "I want to learn everything I can about it. Why it came to me matters as much as the land itself.", scores: { Seeker: 8, Watcher: 3, Challenger: 2, Strategist: 1 } },
          { text: "I start thinking about who else might benefit from it. Land affects more than just its owner.", scores: { Reconciler: 6, Comforter: 4, Steward: 3, Performer: 1 } },
          { text: "I wonder why it was abandoned, neglected, or available in the first place.", scores: { Challenger: 7, Firebrand: 4, Watcher: 2, Seeker: 1 } }
        ]
      },
      {
        text: "You discover an old journal filled with notes from someone you've never met. What is your first thought?",
        maxSelections: 2,
        answers: [
          { text: "I want to know who this person was.", scores: { Seeker: 8, Watcher: 3, Comforter: 1 } },
          { text: "I wonder whether there is something useful inside.", scores: { Builder: 5, Strategist: 3, Architect: 3, Aesthetician: 1 } },
          { text: "I want to understand what kind of life they lived.", scores: { Comforter: 6, Reconciler: 3, Aesthetician: 2 } },
          { text: "I start looking for patterns, inconsistencies, and unanswered questions.", scores: { Watcher: 7, Challenger: 3, Seeker: 2 } },
          { text: "I wonder why this journal survived when so much else did not.", scores: { Aesthetician: 7, Architect: 2, Seeker: 2 } }
        ]
      },
      {
        text: "Two people you care about are arguing. What is your first thought?",
        maxSelections: 2,
        answers: [
          { text: "Something important is being misunderstood.", scores: { Reconciler: 8, Comforter: 3, Watcher: 2 } },
          { text: "I want to know what started this.", scores: { Seeker: 6, Watcher: 4, Challenger: 2 } },
          { text: "If nobody steps in, this may get worse.", scores: { Guardian: 7, Steward: 3, Strategist: 2 } },
          { text: "I wonder whether they're actually arguing about the real problem.", scores: { Challenger: 7, Firebrand: 4, Watcher: 2 } },
          { text: "I wonder what happens after this argument is over.", scores: { Strategist: 7, Architect: 3, Steward: 2 } }
        ]
      },
      {
        text: "You are unexpectedly placed in charge of a project. What is your first thought?",
        maxSelections: 2,
        answers: [
          { text: "What needs to happen first?", scores: { Catalyst: 8, Builder: 2, Strategist: 1 } },
          { text: "What resources and people do I have available?", scores: { Builder: 5, Strategist: 4, Steward: 2, Architect: 2 } },
          { text: "What is the goal, really?", scores: { Architect: 7, Challenger: 2, Strategist: 2 } },
          { text: "Who is depending on this project succeeding?", scores: { Steward: 7, Comforter: 3, Guardian: 2 } },
          { text: "How do I get everyone moving in the same direction?", scores: { Performer: 7, Catalyst: 5, Reconciler: 3 } }
        ]
      },
      {
        text: "You discover you've been wrong about something important. What is your first thought?",
        maxSelections: 2,
        answers: [
          { text: "Well... I'd rather know the truth than keep being wrong.", scores: { Challenger: 8, Seeker: 3, Watcher: 2 } },
          { text: "I need to understand how I got this wrong in the first place.", scores: { Watcher: 7, Strategist: 3, Architect: 2 } },
          { text: "Who else is affected by this?", scores: { Comforter: 6, Steward: 4, Reconciler: 2 } },
          { text: "Alright. What do I need to do now?", scores: { Catalyst: 8, Builder: 3, Firebrand: 2, Guardian: 1 } },
          { text: "I wonder what else I've assumed that deserves another look.", scores: { Seeker: 7, Challenger: 3, Architect: 2 } }
        ]
      },
      {
        text: "A newcomer joins your group. What is your first thought?",
        maxSelections: 2,
        answers: [
          { text: "I hope they feel welcome.", scores: { Comforter: 7, Reconciler: 4, Steward: 2 } },
          { text: "I wonder what they're like.", scores: { Seeker: 7, Watcher: 3, Performer: 1 } },
          { text: "I wonder what they bring that we don't already have.", scores: { Strategist: 6, Architect: 4, Builder: 2, Performer: 2 } },
          { text: "I wonder how they'll change the group.", scores: { Strategist: 7, Watcher: 3, Architect: 2 } },
          { text: "I wonder whether they actually want to be here.", scores: { Challenger: 5, Watcher: 4, Comforter: 2, Reconciler: 1 } },
          { text: "Fresh eyes usually notice things everyone else has stopped seeing.", scores: { Firebrand: 7, Challenger: 3, Seeker: 2 } }
        ]
      },
      {
        text: "Something important breaks. What is your first thought?",
        maxSelections: 2,
        answers: [
          { text: "Can it be fixed?", scores: { Builder: 7, Steward: 3, Architect: 2, Catalyst: 1 } },
          { text: "What depends on it?", scores: { Steward: 7, Guardian: 4, Strategist: 2 } },
          { text: "How did this happen?", scores: { Watcher: 7, Challenger: 3, Seeker: 2 } },
          { text: "Who's taking this the hardest?", scores: { Comforter: 7, Reconciler: 3, Guardian: 1 } },
          { text: "If it broke, maybe it wasn't working as well as everyone thought.", scores: { Firebrand: 7, Challenger: 4, Architect: 1 } },
          { text: "Standing around won't fix it. Someone needs to start moving.", scores: { Catalyst: 8, Builder: 2, Guardian: 1 } }
        ]
      },
      {
        text: "You are invited to a large gathering where you know almost nobody. What is your first thought?",
        maxSelections: 2,
        answers: [
          { text: "I wonder who I'm going to meet.", scores: { Seeker: 6, Performer: 4, Watcher: 2 } },
          { text: "I wonder how this group fits together.", scores: { Watcher: 7, Strategist: 3, Architect: 2 } },
          { text: "There are probably some interesting conversations waiting to happen.", scores: { Performer: 8, Reconciler: 3, Catalyst: 4 } },
          { text: "I hope nobody ends up standing alone.", scores: { Comforter: 6, Reconciler: 4, Steward: 2 } },
          { text: "Someone had a reason for bringing all these people together.", scores: { Architect: 6, Strategist: 4, Challenger: 2 } },
          { text: "Every room has its own energy. I'm curious what this one feels like.", scores: { Performer: 8, Aesthetician: 2, Seeker: 1 } }
        ]
      },
      {
        text: "A long-standing tradition is being abandoned. What is your first thought?",
        maxSelections: 2,
        answers: [
          { text: "Maybe it served a purpose people have forgotten.", scores: { Steward: 7, Architect: 3, Watcher: 2 } },
          { text: "If nobody wants it anymore, perhaps it is time for something new.", scores: { Firebrand: 7, Catalyst: 3, Challenger: 2 } },
          { text: "I want to know why people cared about it in the first place.", scores: { Seeker: 7, Aesthetician: 3, Watcher: 2 } },
          { text: "Some traditions are worth preserving.", scores: { Guardian: 6, Steward: 4, Reconciler: 1 } },
          { text: "I wonder what people are going to gather around instead.", scores: { Performer: 5, Catalyst: 4, Reconciler: 3 } }
        ]
      },
      {
        text: "You discover a secret. What is your first thought?",
        maxSelections: 2,
        answers: [
          { text: "Is it actually true?", scores: { Challenger: 8, Watcher: 3, Seeker: 2 } },
          { text: "Who already knows?", scores: { Strategist: 7, Watcher: 3, Guardian: 2 } },
          { text: "Who could be hurt by this?", scores: { Comforter: 7, Guardian: 3, Steward: 2 } },
          { text: "How did this stay hidden?", scores: { Seeker: 7, Watcher: 4, Architect: 1 } },
          { text: "What changes if people learn about it?", scores: { Firebrand: 7, Catalyst: 5, Strategist: 3 } }
        ]
      },
      {
        text: "You encounter something that moves people deeply. What is your first thought?",
        maxSelections: 2,
        answers: [
          { text: "What makes this beautiful to people?", scores: { Aesthetician: 9, Seeker: 2, Watcher: 1 } },
          { text: "Someone put a lot of care into this.", scores: { Builder: 3, Steward: 4, Aesthetician: 5 } },
          { text: "I wonder what story is attached to it.", scores: { Seeker: 5, Aesthetician: 6, Watcher: 2 } },
          { text: "Beauty tends to bring people together.", scores: { Reconciler: 4, Performer: 5, Aesthetician: 3, Comforter: 2 } },
          { text: "I wonder what people overlook while they're looking at it.", scores: { Challenger: 4, Watcher: 3, Firebrand: 4 } },
          { text: "Beauty usually reveals something deeper than itself.", scores: { Aesthetician: 8, Architect: 3, Seeker: 2 } }
        ]
      },
      {
        text: "You are given the chance to change one thing about the world. What is your first thought?",
        maxSelections: 2,
        answers: [
          { text: "Who needs the help most?", scores: { Comforter: 7, Guardian: 3, Reconciler: 2 } },
          { text: "What change would do the most good over time?", scores: { Architect: 6, Strategist: 5, Steward: 2 } },
          { text: "What problem has everyone simply accepted?", scores: { Firebrand: 7, Challenger: 4, Watcher: 1 } },
          { text: "What change would help people work together better?", scores: { Reconciler: 6, Performer: 4, Catalyst: 4 } },
          { text: "What becomes possible afterward?", scores: { Builder: 3, Catalyst: 5, Seeker: 3 } }
        ]
      },
      {
        text: "Before seeing your result, which of these feels most important to you?",
        maxSelections: 1,
        answers: [
          { text: "Action", selfGuess: "Catalyst", scores: {} },
          { text: "Building", selfGuess: "Builder", scores: {} },
          { text: "Understanding", selfGuess: "Seeker", scores: {} },
          { text: "Truth", selfGuess: "Challenger", scores: {} },
          { text: "Helping", selfGuess: "Comforter", scores: {} },
          { text: "Safety", selfGuess: "Guardian", scores: {} },
          { text: "Beauty", selfGuess: "Aesthetician", scores: {} },
          { text: "Connection", selfGuess: "Reconciler", scores: {} },
          { text: "Planning", selfGuess: "Strategist", scores: {} },
          { text: "Design", selfGuess: "Architect", scores: {} },
          { text: "Stewardship", selfGuess: "Steward", scores: {} },
          { text: "Expression", selfGuess: "Performer", scores: {} },
          { text: "Change", selfGuess: "Firebrand", scores: {} },
          { text: "Awareness", selfGuess: "Watcher", scores: {} }
        ]
      }
    ]
  };
})();
