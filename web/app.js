/* The page owns no tutoring logic (workingReadme 3): it renders the
   server-owned curriculum, sends only the learner's name, chosen
   subject/topic/grade band, and the learner's own words to the API, and shows
   the prompts that come back. After the diagnostic explanation it fetches and
   renders the learner/parent gap report (the funnel's value moment). */
(function () {
  'use strict';

  var startForm = document.getElementById('start-form');
  var chatForm = document.getElementById('chat-form');
  var learnerInput = document.getElementById('learner-name');
  var gradeSelect = document.getElementById('grade-band');
  var subjectSelect = document.getElementById('subject');
  var topicSelect = document.getElementById('topic');
  var beginButton = document.getElementById('begin-button');
  var answerInput = document.getElementById('answer');
  var transcript = document.getElementById('transcript');
  var reportEl = document.getElementById('report');

  var curriculum = null; // { subjects: [{name, topics}], gradeBands: [...] }
  var learner = '';
  // True from begin until the learner has submitted their diagnostic
  // explanation — the point at which the gap report becomes available.
  var awaitingDiagnosis = false;

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

  function handleTurn(data) {
    if (data && data.prompt) {
      addMessage(data.prompt);
    }
    if (data.state === 'mastered') {
      stopTutoring('You have finished this topic. Reload with a new name to start again.');
    }
  }

  function option(value) {
    var el = document.createElement('option');
    el.value = value;
    el.textContent = value;
    return el;
  }

  function populateTopics() {
    topicSelect.innerHTML = '';
    var subject = curriculum.subjects.find(function (s) { return s.name === subjectSelect.value; });
    if (!subject) {
      return;
    }
    subject.topics.forEach(function (topic) {
      topicSelect.appendChild(option(topic));
    });
  }

  function loadCurriculum() {
    beginButton.disabled = true;
    return fetch('/curriculum')
      .then(function (res) {
        if (!res.ok) {
          throw new Error('curriculum request failed');
        }
        return res.json();
      })
      .then(function (data) {
        curriculum = data;
        gradeSelect.innerHTML = '';
        curriculum.gradeBands.forEach(function (band) {
          gradeSelect.appendChild(option(band));
        });
        subjectSelect.innerHTML = '';
        curriculum.subjects.forEach(function (subject) {
          subjectSelect.appendChild(option(subject.name));
        });
        populateTopics();
        beginButton.disabled = false;
      })
      .catch(function () {
        addMessage('Could not load the subject list. Is the server running?', 'system');
      });
  }

  subjectSelect.addEventListener('change', populateTopics);

  // heading/labels are the only text built from server/learner data that we
  // put in the DOM; everything is set with textContent (never innerHTML), so
  // learner- or model-provided text can never become markup.
  function addTextList(container, items) {
    if (!items || items.length === 0) {
      return;
    }
    var ul = document.createElement('ul');
    items.forEach(function (item) {
      var li = document.createElement('li');
      li.textContent = item;
      ul.appendChild(li);
    });
    container.appendChild(ul);
  }

  function renderReport(data) {
    reportEl.innerHTML = '';
    var h2 = document.createElement('h2');
    h2.textContent = 'Your diagnosis';
    reportEl.appendChild(h2);

    var finding = document.createElement('p');
    finding.className = 'finding';
    finding.textContent = 'You told us: ' + (data.finding || '');
    reportEl.appendChild(finding);

    if (data.gaps && data.gaps.length) {
      var h3 = document.createElement('h3');
      h3.textContent = 'What we found';
      reportEl.appendChild(h3);
      addTextList(reportEl, data.gaps);
    }
    if (data.misconceptions && data.misconceptions.length) {
      var h4 = document.createElement('h3');
      h4.textContent = 'A misconception to work on';
      reportEl.appendChild(h4);
      addTextList(reportEl, data.misconceptions);
    }
    if (data.recommendedAction) {
      var rec = document.createElement('p');
      rec.textContent = 'Suggested next step: ' + data.recommendedAction;
      reportEl.appendChild(rec);
    }
    reportEl.hidden = false;
  }

  function showDiagnosisReport() {
    fetch(learnerPath() + '/diagnosis')
      .then(function (res) {
        if (!res.ok) {
          throw new Error('diagnosis report request failed');
        }
        return res.json();
      })
      .then(renderReport)
      .catch(function () { /* report is a nice-to-have; the session continues */ });
  }

  function begin() {
    beginButton.disabled = true;
    postJSON(learnerPath() + '/begin', {
      subject: subjectSelect.value,
      topic: topicSelect.value,
      gradeBand: gradeSelect.value,
    }).then(function (result) {
      beginButton.disabled = false;
      if (!result.ok) {
        addMessage('Could not start: ' + (result.data.error || result.status), 'system');
        return;
      }
      startForm.hidden = true;
      reportEl.hidden = true;
      chatForm.hidden = false;
      awaitingDiagnosis = result.data.state === 'diagnosing';
      answerInput.focus();
      handleTurn(result.data);
    }).catch(function () {
      beginButton.disabled = false;
      addMessage('Could not reach the server. Is it running?', 'system');
    });
  }

  function sendAnswer(text) {
    postJSON(learnerPath() + '/input', { input: text }).then(function (result) {
      if (result.ok) {
        if (awaitingDiagnosis) {
          awaitingDiagnosis = false;
          showDiagnosisReport();
        }
        handleTurn(result.data);
        return;
      }
      if (result.status === 409) {
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
    reportEl.hidden = true;
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

  loadCurriculum();
})();
