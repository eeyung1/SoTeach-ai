/* The page owns no tutoring logic (workingReadme 3): it only sends the
   learner's name, the chosen topic, and the learner's own words to the API,
   and shows the prompts that come back. */
(function () {
  'use strict';

  var SUBJECT = 'Mathematics';
  var TOPIC = 'Addition';

  var startForm = document.getElementById('start-form');
  var chatForm = document.getElementById('chat-form');
  var learnerInput = document.getElementById('learner-name');
  var answerInput = document.getElementById('answer');
  var transcript = document.getElementById('transcript');

  var learner = '';

  function learnerPath() {
    return '/learners/' + encodeURIComponent(learner);
  }

  function addMessage(text, kind) {
    transcript.hidden = false;
    var div = document.createElement('div');
    div.className = 'message ' + (kind || 'tutor');
    div.textContent = text;
    transcript.appendChild(div);
    transcript.scrollTop = transcript.scrollHeight;
  }

  function postJSON(path, body) {
    return fetch(path, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    }).then(function (res) {
      return res.json().catch(function () { return {}; }).then(function (data) {
        return { ok: res.ok, status: res.status, data: data };
      });
    });
  }

  function stopTutoring(reason) {
    chatForm.querySelector('button').disabled = true;
    answerInput.disabled = true;
    if (reason) {
      addMessage(reason, 'system');
    }
  }

  // handleTurn shows whatever prompt the server returned and decides whether
  // the session still expects input.
  function handleTurn(data) {
    if (data && data.prompt) {
      addMessage(data.prompt);
    }
    if (data.state === 'mastered') {
      stopTutoring('You have finished this topic. Reload with a new name to start again.');
    }
  }

  function begin() {
    postJSON(learnerPath() + '/begin', { subject: SUBJECT, topic: TOPIC }).then(function (result) {
      if (!result.ok) {
        addMessage('Could not start: ' + (result.data.error || result.status), 'system');
        startForm.hidden = false;
        return;
      }
      chatForm.hidden = false;
      answerInput.focus();
      handleTurn(result.data);
    }).catch(function () {
      addMessage('Could not reach the server. Is it running?', 'system');
    });
  }

  function sendAnswer(text) {
    postJSON(learnerPath() + '/input', { input: text }).then(function (result) {
      if (result.ok) {
        handleTurn(result.data);
        return;
      }
      if (result.status === 409) {
        // Nothing awaiting input (e.g. already mastered): end politely.
        stopTutoring('There is nothing to answer right now. Reload with a new name to start again.');
        return;
      }
      addMessage('Something went wrong: ' + (result.data.error || result.status), 'system');
    }).catch(function () {
      addMessage('Could not reach the server. Is it running?', 'system');
    });
  }

  startForm.addEventListener('submit', function (event) {
    event.preventDefault();
    learner = learnerInput.value.trim();
    if (!learner) {
      return;
    }
    startForm.hidden = true;
    transcript.innerHTML = '';
    addMessage('Hi ' + learner + '! The tutor will ask you questions now.', 'system');
    begin();
  });

  chatForm.addEventListener('submit', function (event) {
    event.preventDefault();
    var text = answerInput.value.trim();
    if (!text) {
      return;
    }
    answerInput.value = '';
    sendAnswer(text);
  });
})();
