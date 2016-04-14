;;; smart-mode.el --- smart file editing commands for Emacs -*- lexical-binding:t -*-

;; Copyright (C) 2016 Duzy Chan <code@duzy.info>, http://duzy.info

;; Author: Duzy Chan <code@duzy.info>
;; Maintainer: code@duzy.info
;; Keywords: unix, tools
(require 'make-mode)

(defgroup smart nil
  "Smart editing commands for Emacs."
  :link '(custom-group-link :tag "Font Lock Faces group" font-lock-faces)
  :group 'tools
  :prefix "smart-")

(defface smart-module-name-face
  ;; This needs to go along both with foreground and background colors (i.e. shell)
  '((t (:inherit font-lock-variable-name-face))) ;; (:background  "LightBlue1")
  "Face to use for additionally highlighting rule targets in Font-Lock mode."
  :group 'smart
  :version "22.1")

(defconst smart-var-use-regex
  "[^$]\\$[({]\\([-a-zA-Z0-9_.]+\\|[@%<?^+*][FD]?\\)"
  "Regex used to find $(macro) uses in a makefile.")

(defconst smart-statements
  `("include" "template" "module" "commit" "use" "post"
    ,@(cdr makefile-statements))
  "List of keywords understood by smart.")

(defconst smart-font-lock-keywords
  (makefile-make-font-lock-keywords
   smart-var-use-regex
   smart-statements
   t
   "^\\(?: [ \t]*\\)?if\\(n\\)\\(?:def\\|eq\\)\\>"

   '("[^$]\\(\\$[({][@%*][DF][})]\\)"
     1 'makefile-targets append)

   ;; $(function ...) ${function ...}
   '("[^$]\\$[({]\\([-a-zA-Z0-9_.]+\\s \\)"
     1 font-lock-function-name-face prepend)

   ;; $(shell ...) ${shell ...}
   '("[^$]\\$\\([({]\\)shell[ \t]+"
     makefile-match-function-end nil nil
     (1 'makefile-shell prepend t))

   ;; $(template ...) $(module ...)
   '("[^$]\\$[({]\\(template\\|module\\|use\\)[ \t]\\([^,)}]+\\)"
     (1 font-lock-builtin-face prepend)
     (2 font-lock-string-face prepend))

   ;; $(commit ...)
   '("[^$]\\$[({]\\(commit\\|post\\)[ \t)}]"
     1 font-lock-builtin-face prepend)
   ))

(define-derived-mode smart-mode makefile-gmake-mode "smart"
  "Major mode for editing .smart files."
  (setq font-lock-defaults
	`(smart-font-lock-keywords ,@(cdr font-lock-defaults))))


(progn (add-to-list 'auto-mode-alist '("\\.smart" . smart-mode))
       (message "smart-mode"))
(provide 'smart-mode)
