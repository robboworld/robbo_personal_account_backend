#!/usr/bin/env python3
"""Patch graph/generated/generated.go for profile fields (gqlgen unavailable in CI)."""
from pathlib import Path

GEN = Path(__file__).resolve().parents[1] / "graph/generated/generated.go"
text = GEN.read_text()

OLD_SCHEMA_USER = """type UserHttp {
    id: String!
    email: String!
    password: String!
    role: Int!
    nickname: String!
    firstname: String!
    lastname: String!
    middlename: String!
    createdAt: Timestamp!
}

input UpdateProfileInput {
    id: String!
    email: String!
    nickname: String!
    firstname: String!
    lastname: String!
    middlename: String!
}"""

NEW_SCHEMA_USER = """type UserHttp {
    id: String!
    email: String!
    password: String!
    role: Int!
    nickname: String!
    fullName: String!
    firstname: String!
    lastname: String!
    middlename: String!
    levelOfEducation: String
    country: String
    yearOfBirth: Int
    gender: String
    language: String
    createdAt: Timestamp!
}

input UpdateProfileInput {
    id: String!
    email: String!
    nickname: String!
    fullName: String!
    firstname: String!
    lastname: String!
    middlename: String!
    levelOfEducation: String
    country: String
    yearOfBirth: Int
    gender: String
    language: String
}"""

if OLD_SCHEMA_USER not in text:
    raise SystemExit("embedded schema block not found")
text = text.replace(OLD_SCHEMA_USER, NEW_SCHEMA_USER)

OLD_UNMARSHAL_ORDER = 'fieldsInOrder := [...]string{"id", "email", "nickname", "firstname", "lastname", "middlename"}'
NEW_UNMARSHAL_ORDER = 'fieldsInOrder := [...]string{"id", "email", "nickname", "fullName", "firstname", "lastname", "middlename", "levelOfEducation", "country", "yearOfBirth", "gender", "language"}'
text = text.replace(OLD_UNMARSHAL_ORDER, NEW_UNMARSHAL_ORDER, 1)

INSERT_AFTER_NICKNAME = """\tcase "nickname":
\t\tvar err error

\t\tctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("nickname"))
\t\tit.Nickname, err = ec.unmarshalNString2string(ctx, v)
\t\tif err != nil {
\t\t\treturn it, err
\t\t}
\tcase "fullName":"""

INSERT_FULLNAME_CASE = """\tcase "fullName":
\t\tvar err error

\t\tctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("fullName"))
\t\tit.FullName, err = ec.unmarshalNString2string(ctx, v)
\t\tif err != nil {
\t\t\treturn it, err
\t\t}"""

# insert fullName case after nickname case in unmarshalInputUpdateProfileInput only
idx = text.find("func (ec *executionContext) unmarshalInputUpdateProfileInput")
if idx == -1:
    raise SystemExit("unmarshalInputUpdateProfileInput not found")
nick = text.find('case "nickname":', idx)
if nick == -1:
    raise SystemExit("nickname case not found")
end_nick = text.find('case "firstname":', nick)
if 'case "fullName":' not in text[idx:end_nick + 200]:
    text = text[:end_nick] + INSERT_FULLNAME_CASE + "\n" + text[end_nick:]

EXTRA_CASES = """
\tcase "levelOfEducation":
\t\tvar err error

\t\tctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("levelOfEducation"))
\t\tit.LevelOfEducation, err = ec.unmarshalOString2ᚖstring(ctx, v)
\t\tif err != nil {
\t\t\treturn it, err
\t\t}
\tcase "country":
\t\tvar err error

\t\tctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("country"))
\t\tit.Country, err = ec.unmarshalOString2ᚖstring(ctx, v)
\t\tif err != nil {
\t\t\treturn it, err
\t\t}
\tcase "yearOfBirth":
\t\tvar err error

\t\tctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("yearOfBirth"))
\t\tit.YearOfBirth, err = ec.unmarshalOInt2ᚖint(ctx, v)
\t\tif err != nil {
\t\t\treturn it, err
\t\t}
\tcase "gender":
\t\tvar err error

\t\tctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("gender"))
\t\tit.Gender, err = ec.unmarshalOString2ᚖstring(ctx, v)
\t\tif err != nil {
\t\t\treturn it, err
\t\t}
\tcase "language":
\t\tvar err error

\t\tctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("language"))
\t\tit.Language, err = ec.unmarshalOString2ᚖstring(ctx, v)
\t\tif err != nil {
\t\t\treturn it, err
\t\t}"""

mid = text.find('case "middlename":', idx)
end_mid = text.find("\n\t}\n\n\treturn it, nil\n}", mid)
if 'case "levelOfEducation":' not in text[idx:idx + 3000]:
    text = text[:end_mid] + EXTRA_CASES + text[end_mid:]

FIELD_CTX_SNIPPET = """
\t\t\tcase "fullName":
\t\t\t\treturn ec.fieldContext_UserHttp_fullName(ctx, field)
\t\t\tcase "levelOfEducation":
\t\t\t\treturn ec.fieldContext_UserHttp_levelOfEducation(ctx, field)
\t\t\tcase "country":
\t\t\t\treturn ec.fieldContext_UserHttp_country(ctx, field)
\t\t\tcase "yearOfBirth":
\t\t\t\treturn ec.fieldContext_UserHttp_yearOfBirth(ctx, field)
\t\t\tcase "gender":
\t\t\t\treturn ec.fieldContext_UserHttp_gender(ctx, field)
\t\t\tcase "language":
\t\t\t\treturn ec.fieldContext_UserHttp_language(ctx, field)"""

needle = '\t\t\tcase "nickname":\n\t\t\t\treturn ec.fieldContext_UserHttp_nickname(ctx, field)\n\t\t\tcase "firstname":'
if needle in text and 'fieldContext_UserHttp_fullName' not in text:
    text = text.replace(
        needle,
        '\t\t\tcase "nickname":\n\t\t\t\treturn ec.fieldContext_UserHttp_nickname(ctx, field)'
        + FIELD_CTX_SNIPPET
        + '\n\t\t\tcase "firstname":',
    )

USERHTTP_RESOLVERS = '''
func (ec *executionContext) _UserHttp_fullName(ctx context.Context, field graphql.CollectedField, obj *models.UserHTTP) (ret graphql.Marshaler) {
	fc, err := ec.fieldContext_UserHttp_fullName(ctx, field)
	if err != nil {
		return graphql.Null
	}
	ctx = graphql.WithFieldContext(ctx, fc)
	defer func() {
		if r := recover(); r != nil {
			ec.Error(ctx, ec.Recover(ctx, r))
			ret = graphql.Null
		}
	}()
	resTmp, err := ec.ResolverMiddleware(ctx, func(rctx context.Context) (interface{}, error) {
		ctx = rctx
		return obj.FullName, nil
	})
	if err != nil {
		ec.Error(ctx, err)
		return graphql.Null
	}
	if resTmp == nil {
		if !graphql.HasFieldError(ctx, fc) {
			ec.Errorf(ctx, "must not be null")
		}
		return graphql.Null
	}
	res := resTmp.(string)
	fc.Result = res
	return ec.marshalNString2string(ctx, field.Selections, res)
}

func (ec *executionContext) fieldContext_UserHttp_fullName(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "UserHttp",
		Field:      field,
		IsMethod:   false,
		IsResolver: false,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return nil, errors.New("field of type String does not have child fields")
		},
	}
	return fc, nil
}

func (ec *executionContext) _UserHttp_levelOfEducation(ctx context.Context, field graphql.CollectedField, obj *models.UserHTTP) (ret graphql.Marshaler) {
	fc, err := ec.fieldContext_UserHttp_levelOfEducation(ctx, field)
	if err != nil {
		return graphql.Null
	}
	ctx = graphql.WithFieldContext(ctx, fc)
	defer func() {
		if r := recover(); r != nil {
			ec.Error(ctx, ec.Recover(ctx, r))
			ret = graphql.Null
		}
	}()
	resTmp, err := ec.ResolverMiddleware(ctx, func(rctx context.Context) (interface{}, error) {
		ctx = rctx
		return obj.LevelOfEducation, nil
	})
	if err != nil {
		ec.Error(ctx, err)
		return graphql.Null
	}
	if resTmp == nil {
		return graphql.Null
	}
	res := resTmp.(*string)
	fc.Result = res
	return ec.marshalOString2ᚖstring(ctx, field.Selections, res)
}

func (ec *executionContext) fieldContext_UserHttp_levelOfEducation(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "UserHttp",
		Field:      field,
		IsMethod:   false,
		IsResolver: false,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return nil, errors.New("field of type String does not have child fields")
		},
	}
	return fc, nil
}

func (ec *executionContext) _UserHttp_country(ctx context.Context, field graphql.CollectedField, obj *models.UserHTTP) (ret graphql.Marshaler) {
	fc, err := ec.fieldContext_UserHttp_country(ctx, field)
	if err != nil {
		return graphql.Null
	}
	ctx = graphql.WithFieldContext(ctx, fc)
	defer func() {
		if r := recover(); r != nil {
			ec.Error(ctx, ec.Recover(ctx, r))
			ret = graphql.Null
		}
	}()
	resTmp, err := ec.ResolverMiddleware(ctx, func(rctx context.Context) (interface{}, error) {
		ctx = rctx
		return obj.Country, nil
	})
	if err != nil {
		ec.Error(ctx, err)
		return graphql.Null
	}
	if resTmp == nil {
		return graphql.Null
	}
	res := resTmp.(*string)
	fc.Result = res
	return ec.marshalOString2ᚖstring(ctx, field.Selections, res)
}

func (ec *executionContext) fieldContext_UserHttp_country(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "UserHttp",
		Field:      field,
		IsMethod:   false,
		IsResolver: false,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return nil, errors.New("field of type String does not have child fields")
		},
	}
	return fc, nil
}

func (ec *executionContext) _UserHttp_yearOfBirth(ctx context.Context, field graphql.CollectedField, obj *models.UserHTTP) (ret graphql.Marshaler) {
	fc, err := ec.fieldContext_UserHttp_yearOfBirth(ctx, field)
	if err != nil {
		return graphql.Null
	}
	ctx = graphql.WithFieldContext(ctx, fc)
	defer func() {
		if r := recover(); r != nil {
			ec.Error(ctx, ec.Recover(ctx, r))
			ret = graphql.Null
		}
	}()
	resTmp, err := ec.ResolverMiddleware(ctx, func(rctx context.Context) (interface{}, error) {
		ctx = rctx
		return obj.YearOfBirth, nil
	})
	if err != nil {
		ec.Error(ctx, err)
		return graphql.Null
	}
	if resTmp == nil {
		return graphql.Null
	}
	res := resTmp.(*int)
	fc.Result = res
	return ec.marshalOInt2ᚖint(ctx, field.Selections, res)
}

func (ec *executionContext) fieldContext_UserHttp_yearOfBirth(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "UserHttp",
		Field:      field,
		IsMethod:   false,
		IsResolver: false,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return nil, errors.New("field of type Int does not have child fields")
		},
	}
	return fc, nil
}

func (ec *executionContext) _UserHttp_gender(ctx context.Context, field graphql.CollectedField, obj *models.UserHTTP) (ret graphql.Marshaler) {
	fc, err := ec.fieldContext_UserHttp_gender(ctx, field)
	if err != nil {
		return graphql.Null
	}
	ctx = graphql.WithFieldContext(ctx, fc)
	defer func() {
		if r := recover(); r != nil {
			ec.Error(ctx, ec.Recover(ctx, r))
			ret = graphql.Null
		}
	}()
	resTmp, err := ec.ResolverMiddleware(ctx, func(rctx context.Context) (interface{}, error) {
		ctx = rctx
		return obj.Gender, nil
	})
	if err != nil {
		ec.Error(ctx, err)
		return graphql.Null
	}
	if resTmp == nil {
		return graphql.Null
	}
	res := resTmp.(*string)
	fc.Result = res
	return ec.marshalOString2ᚖstring(ctx, field.Selections, res)
}

func (ec *executionContext) fieldContext_UserHttp_gender(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "UserHttp",
		Field:      field,
		IsMethod:   false,
		IsResolver: false,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return nil, errors.New("field of type String does not have child fields")
		},
	}
	return fc, nil
}

func (ec *executionContext) _UserHttp_language(ctx context.Context, field graphql.CollectedField, obj *models.UserHTTP) (ret graphql.Marshaler) {
	fc, err := ec.fieldContext_UserHttp_language(ctx, field)
	if err != nil {
		return graphql.Null
	}
	ctx = graphql.WithFieldContext(ctx, fc)
	defer func() {
		if r := recover(); r != nil {
			ec.Error(ctx, ec.Recover(ctx, r))
			ret = graphql.Null
		}
	}()
	resTmp, err := ec.ResolverMiddleware(ctx, func(rctx context.Context) (interface{}, error) {
		ctx = rctx
		return obj.Language, nil
	})
	if err != nil {
		ec.Error(ctx, err)
		return graphql.Null
	}
	if resTmp == nil {
		return graphql.Null
	}
	res := resTmp.(*string)
	fc.Result = res
	return ec.marshalOString2ᚖstring(ctx, field.Selections, res)
}

func (ec *executionContext) fieldContext_UserHttp_language(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "UserHttp",
		Field:      field,
		IsMethod:   false,
		IsResolver: false,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return nil, errors.New("field of type String does not have child fields")
		},
	}
	return fc, nil
}

'''

if "_UserHttp_fullName" not in text:
    anchor = "func (ec *executionContext) _UserHttp_firstname("
    text = text.replace(anchor, USERHTTP_RESOLVERS + anchor, 1)

MARSHAL_NICK = """\t\tcase "nickname":

\t\t\tout.Values[i] = ec._UserHttp_nickname(ctx, field, obj)

\t\t\tif out.Values[i] == graphql.Null {
\t\t\t\tinvalids++
\t\t\t}
\t\tcase "firstname":"""

MARSHAL_NEW = """\t\tcase "nickname":

\t\t\tout.Values[i] = ec._UserHttp_nickname(ctx, field, obj)

\t\t\tif out.Values[i] == graphql.Null {
\t\t\t\tinvalids++
\t\t\t}
\t\tcase "fullName":

\t\t\tout.Values[i] = ec._UserHttp_fullName(ctx, field, obj)

\t\t\tif out.Values[i] == graphql.Null {
\t\t\t\tinvalids++
\t\t\t}
\t\tcase "levelOfEducation":

\t\t\tout.Values[i] = ec._UserHttp_levelOfEducation(ctx, field, obj)
\t\tcase "country":

\t\t\tout.Values[i] = ec._UserHttp_country(ctx, field, obj)
\t\tcase "yearOfBirth":

\t\t\tout.Values[i] = ec._UserHttp_yearOfBirth(ctx, field, obj)
\t\tcase "gender":

\t\t\tout.Values[i] = ec._UserHttp_gender(ctx, field, obj)
\t\tcase "language":

\t\t\tout.Values[i] = ec._UserHttp_language(ctx, field, obj)
\t\tcase "firstname":"""

if MARSHAL_NICK in text and 'case "fullName":' not in text.split("func (ec *executionContext) _UserHttp(ctx context.Context")[1].split("case \"createdAt\":")[0]:
    text = text.replace(MARSHAL_NICK, MARSHAL_NEW, 1)

GEN.write_text(text)
print("patched", GEN)
