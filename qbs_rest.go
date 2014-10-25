package seven5

import (
	"github.com/coocood/qbs"
)

//QbsRestIndex is the QBS version of RestIndex
type QbsRestIndex interface {
	IndexQbs(PBundle, *qbs.Qbs) (interface{}, error)
}

//QbsRestFind is the QBS version of RestFind
type QbsRestFind interface {
	FindQbs(int64, PBundle, *qbs.Qbs) (interface{}, error)
}

//QbsRestFindUdid is the QBS version of RestFindUdid
type QbsRestFindUdid interface {
	FindQbs(string, PBundle, *qbs.Qbs) (interface{}, error)
}

//QbsRestDelete is the QBS version of RestDelete
type QbsRestDelete interface {
	DeleteQbs(int64, PBundle, *qbs.Qbs) (interface{}, error)
}

//QbsRestDeleteUdid is the QBS version of RestDeleteUdid
type QbsRestDeleteUdid interface {
	DeleteQbs(string, PBundle, *qbs.Qbs) (interface{}, error)
}

//QbsRestPut is the QBS version RestPut
type QbsRestPut interface {
	PutQbs(int64, interface{}, PBundle, *qbs.Qbs) (interface{}, error)
}

//QbsRestPut is the QBS version RestPutUdid
type QbsRestPutUdid interface {
	PutQbs(string, interface{}, PBundle, *qbs.Qbs) (interface{}, error)
}

//QbsRestPost is the QBS version RestPost
type QbsRestPost interface {
	PostQbs(interface{}, PBundle, *qbs.Qbs) (interface{}, error)
}

//QbsRestAll is the same as RestAll but with the additional qbs.Qbs parameter
//on each method.
type QbsRestAll interface {
	QbsRestIndex
	QbsRestFind
	QbsRestDelete
	QbsRestPost
	QbsRestPut
}

//QbsRestAllUdid is the same as RestAllUdid but with the additional qbs.Qbs parameter
//on each method.
type QbsRestAllUdid interface {
	QbsRestIndex
	QbsRestFindUdid
	QbsRestDeleteUdid
	QbsRestPost
	QbsRestPutUdid
}

//qbsWrapped is just a type for wrapping around qbs-based rest methods that
//want to "appear" as simple rest methods.  Note that this is type safe and
//there is no worry about nil values if you use the QbsWrap* methods.
type qbsWrapped struct {
	store *QbsStore
	index QbsRestIndex
	find  QbsRestFind
	del   QbsRestDelete
	put   QbsRestPut
	post  QbsRestPost
}

type qbsWrappedUdid struct {
	store *QbsStore
	index QbsRestIndex
	find  QbsRestFindUdid
	del   QbsRestDeleteUdid
	put   QbsRestPutUdid
	post  QbsRestPost
}

//
// WRAPPED
//

func (self *qbsWrapped) applyPolicy(pb PBundle, fn func(*qbs.Qbs) (interface{}, error)) (result_obj interface{}, result_error error) {
	tx := self.store.Policy.StartTransaction(self.store.Q)
	defer func() {
		if x := recover(); x != nil {
			result_obj, result_error = self.store.Policy.HandlePanic(tx, x)
		}
	}()
	value, err := fn(tx)
	return self.store.Policy.HandleResult(tx, value, err)
}

//Index meets the interface RestIndex but calls the wrapped QBSRestIndex
func (self *qbsWrapped) Index(pb PBundle) (interface{}, error) {
	return self.applyPolicy(pb, func(tx *qbs.Qbs) (interface{}, error) {
		return self.index.IndexQbs(pb, tx)
	})
}

//Find meets the interface RestFind but calls the wrapped QBSRestFind
func (self *qbsWrapped) Find(id int64, pb PBundle) (interface{}, error) {
	return self.applyPolicy(pb, func(tx *qbs.Qbs) (interface{}, error) {
		return self.find.FindQbs(id, pb, tx)
	})
}

//Delete meets the interface RestDelete but calls the wrapped QBSRestDelete
func (self *qbsWrapped) Delete(id int64, pb PBundle) (interface{}, error) {
	return self.applyPolicy(pb, func(tx *qbs.Qbs) (interface{}, error) {
		return self.del.DeleteQbs(id, pb, tx)
	})
}

//Put meets the interface RestPut but calls the wrapped QBSRestPut
func (self *qbsWrapped) Put(id int64, value interface{}, pb PBundle) (interface{}, error) {
	return self.applyPolicy(pb, func(tx *qbs.Qbs) (interface{}, error) {
		return self.put.PutQbs(id, value, pb, tx)
	})
}

//Post meets the interface RestPost but calls the wrapped QBSRestPost
func (self *qbsWrapped) Post(value interface{}, pb PBundle) (interface{}, error) {
	return self.applyPolicy(pb, func(tx *qbs.Qbs) (interface{}, error) {
		return self.post.PostQbs(value, pb, tx)
	})
}

//
// WRAPPED UDID
//

func (self *qbsWrappedUdid) applyPolicy(pb PBundle, fn func(*qbs.Qbs) (interface{}, error)) (result_obj interface{}, result_error error) {
	tx := self.store.Policy.StartTransaction(self.store.Q)
	defer func() {
		if x := recover(); x != nil {
			result_obj, result_error = self.store.Policy.HandlePanic(tx, x)
		}
	}()
	value, err := fn(tx)
	return self.store.Policy.HandleResult(tx, value, err)
}

//Index meets the interface RestIndex but calls the wrapped QBSRestIndex
func (self *qbsWrappedUdid) Index(pb PBundle) (interface{}, error) {
	return self.applyPolicy(pb, func(tx *qbs.Qbs) (interface{}, error) {
		return self.index.IndexQbs(pb, tx)
	})
}

//FindUdid meets the interface RestFindUdid but calls the wrapped QBSRestFindUdid
func (self *qbsWrappedUdid) Find(id string, pb PBundle) (interface{}, error) {
	return self.applyPolicy(pb, func(tx *qbs.Qbs) (interface{}, error) {
		return self.find.FindQbs(id, pb, tx)
	})
}

//DeleteUdid meets the interface RestDeleteUdid but calls the wrapped QBSRestDeleteUdid
func (self *qbsWrappedUdid) Delete(id string, pb PBundle) (interface{}, error) {
	return self.applyPolicy(pb, func(tx *qbs.Qbs) (interface{}, error) {
		return self.del.DeleteQbs(id, pb, tx)
	})
}

//Post meets the interface RestPost but calls the wrapped QBSRestPost
func (self *qbsWrappedUdid) Post(value interface{}, pb PBundle) (interface{}, error) {
	return self.applyPolicy(pb, func(tx *qbs.Qbs) (interface{}, error) {
		return self.post.PostQbs(value, pb, tx)
	})
}

//PutUdid meets the interface RestPutUdid but calls the wrapped QBSRestPutUdid
func (self *qbsWrappedUdid) Put(id string, value interface{}, pb PBundle) (interface{}, error) {
	return self.applyPolicy(pb, func(tx *qbs.Qbs) (interface{}, error) {
		return self.put.PutQbs(id, value, pb, tx)
	})
}

//
// WRAPPING FUNCITONS
//

//Given a QbsRestAll return a RestAll
func QbsWrapAll(a QbsRestAll, s *QbsStore) RestAll {
	return &qbsWrapped{store: s, index: a, find: a, del: a, put: a, post: a}
}

//Given a QbsRestAllUdid return a RestAllUdid
func QbsWrapAllUdid(a QbsRestAllUdid, s *QbsStore) RestAllUdid {
	return &qbsWrappedUdid{store: s, index: a, find: a, del: a, put: a, post: a}
}

//Given a QBSRestIndex return a RestIndex
func QbsWrapIndex(indexer QbsRestIndex, s *QbsStore) RestIndex {
	return &qbsWrapped{index: indexer, store: s}
}

//Given a QbsRestFind return a RestFind
func QbsWrapFind(finder QbsRestFind, s *QbsStore) RestFind {
	return &qbsWrapped{find: finder, store: s}
}

//Given a QbsRestDelete return a RestDelete
func QbsWrapDelete(deler QbsRestDelete, s *QbsStore) RestDelete {
	return &qbsWrapped{del: deler, store: s}
}

//Given a QbsRestPut return a RestPut
func QbsWrapPut(puter QbsRestPut, s *QbsStore) RestPut {
	return &qbsWrapped{put: puter, store: s}
}

//Given a QbsRestPost return a RestPost
func QbsWrapPost(poster QbsRestPost, s *QbsStore) RestPost {
	return &qbsWrapped{post: poster, store: s}
}
